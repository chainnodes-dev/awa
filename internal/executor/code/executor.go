// Package code executes user-written JavaScript inside a secure goja sandbox.
// It is used by the Phaxa "code" state type — the equivalent of n8n's Code node.
//
// Security model (v1 — fully sandboxed):
//   - No network access (no fetch, no XMLHttpRequest)
//   - No filesystem access (no require, no fs)
//   - No access to Go host process (no __proto__ escapes matter here — pure goja)
//   - Execution is bounded by the state's timeout via context cancellation
//
// API contract for user scripts:
//
//	// Style A — imperative (mutate bb directly, call trigger() for early exit)
//	bb.total = bb.value_1 + bb.value_2;
//	trigger("success");
//
//	// Style B — functional (return an object)
//	return {
//	    blackboard_updates: { total: bb.value_1 + bb.value_2 },
//	    trigger: "success",
//	    reasoning: "sum computed"
//	};
//
//	// Styles can be mixed: bb mutations + return trigger.
//	bb.total = bb.value_1 + bb.value_2;
//	return { trigger: bb.total > 100 ? "high" : "normal" };
package code

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// Execute runs the JavaScript code against the given blackboard and returns a
// result compatible with asmtypes.AgentOutput so it slots into the same
// workflow machinery used by agent and script states.
func Execute(code string, bb map[string]interface{}, validTriggers []string, timeout time.Duration, heartbeat func()) (*asmtypes.AgentOutput, error) {
	vm := goja.New()

	// ── Blackboard object ────────────────────────────────────────────────────
	// Build a JS object from the blackboard so mutations in the script are
	// captured when we export it back to Go after execution.
	bbObj := vm.NewObject()
	for k, v := range bb {
		if err := bbObj.Set(k, toJSValue(vm, v)); err != nil {
			return nil, fmt.Errorf("code: failed to inject bb.%s: %w", k, err)
		}
	}
	if err := vm.Set("bb", bbObj); err != nil {
		return nil, fmt.Errorf("code: failed to set bb: %w", err)
	}

	// ── console.{log,warn,error,info} ────────────────────────────────────────
	var consoleLogs []string
	logFn := func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			parts[i] = a.String()
		}
		consoleLogs = append(consoleLogs, strings.Join(parts, " "))
		return goja.Undefined()
	}
	console := vm.NewObject()
	for _, name := range []string{"log", "warn", "error", "info", "debug"} {
		if err := console.Set(name, logFn); err != nil {
			return nil, fmt.Errorf("code: failed to set console.%s: %w", name, err)
		}
	}
	if err := vm.Set("console", console); err != nil {
		return nil, fmt.Errorf("code: failed to set console: %w", err)
	}

	// ── trigger() early-exit function ────────────────────────────────────────
	var (
		earlyTrigger string
		earlyExited  bool
	)
	if err := vm.Set("trigger", func(call goja.FunctionCall) goja.Value {
		earlyTrigger = call.Argument(0).String()
		earlyExited = true
		vm.Interrupt("trigger")
		return goja.Undefined()
	}); err != nil {
		return nil, fmt.Errorf("code: failed to set trigger: %w", err)
	}

	// ── Timeout + Temporal heartbeat ─────────────────────────────────────────
	done := make(chan struct{})
	var interrupted string
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		for {
			select {
			case <-done:
				return
			case <-timer.C:
				interrupted = "timeout"
				vm.Interrupt("timeout")
				return
			case <-ticker.C:
				if heartbeat != nil {
					heartbeat()
				}
			}
		}
	}()
	defer close(done)

	// ── Run ──────────────────────────────────────────────────────────────────
	val, runErr := vm.RunString(wrapCode(code))
	vm.ClearInterrupt()

	if runErr != nil {
		if ie, ok := runErr.(*goja.InterruptedError); ok {
			msg := fmt.Sprintf("%v", ie.Value())
			switch msg {
			case "trigger":
				// Normal path — trigger() was called; fall through.
			case "timeout":
				return nil, fmt.Errorf("code: execution timed out after %s", timeout)
			default:
				if interrupted != "" {
					return nil, fmt.Errorf("code: execution %s", interrupted)
				}
				return nil, fmt.Errorf("code: interrupted: %s", msg)
			}
		} else if je, ok := runErr.(*goja.Exception); ok {
			stack := ""
			if obj := je.Value().ToObject(vm); obj != nil {
				stack = obj.Get("stack").String()
			}
			return nil, fmt.Errorf("code: %v\n\nStack Trace:\n%s", je.Value(), stack)
		} else {
			return nil, fmt.Errorf("code: %w", runErr)
		}
	}

	// ── Collect bb mutations (diff original vs final) ─────────────────────
	bbUpdates := make(map[string]interface{})
	for _, k := range bbObj.Keys() {
		exported := bbObj.Get(k).Export()
		orig, existed := bb[k]
		if !existed || !jsonEqual(orig, exported) {
			bbUpdates[k] = exported
		}
	}

	// ── Determine trigger and reasoning from return value ─────────────────
	var triggerName, reasoning string

	if earlyExited {
		triggerName = earlyTrigger
	} else if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
		exported := val.Export()
		if retMap, ok := exported.(map[string]interface{}); ok {
			if t, ok := retMap["trigger"].(string); ok {
				triggerName = t
			}
			if r, ok := retMap["reasoning"].(string); ok {
				reasoning = r
			}
			// Explicit blackboard_updates override bb mutations (return value wins).
			if u, ok := retMap["blackboard_updates"].(map[string]interface{}); ok {
				for k, v := range u {
					bbUpdates[k] = v
				}
			}
		}
	}

	if triggerName == "" {
		hint := ""
		if len(consoleLogs) > 0 {
			hint = "\nconsole output:\n" + strings.Join(consoleLogs, "\n")
		}
		return nil, fmt.Errorf("code: script must return { trigger: '...' } or call trigger('...')%s", hint)
	}

	// ── Validate trigger ──────────────────────────────────────────────────
	if len(validTriggers) > 0 {
		found := false
		for _, vt := range validTriggers {
			if vt == triggerName {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("code: trigger %q is not valid; expected one of [%s]",
				triggerName, strings.Join(validTriggers, ", "))
		}
	}

	return &asmtypes.AgentOutput{
		BlackboardUpdates: bbUpdates,
		Trigger:           triggerName,
		Reasoning:         reasoning,
	}, nil
}

// wrapCode wraps user code in an IIFE so that `return` works at the top level
// and local `var` declarations don't leak into the global goja scope.
func wrapCode(code string) string {
	return "(function(){\n" + code + "\n})()"
}

// toJSValue converts a Go value to a goja-compatible value, preserving nested
// maps and slices so the full blackboard structure is accessible in JavaScript.
func toJSValue(vm *goja.Runtime, v interface{}) goja.Value {
	// Round-trip through JSON to get a plain JS-compatible value.
	// This handles nested maps/slices correctly without reflection gymnastics.
	b, err := json.Marshal(v)
	if err != nil {
		return vm.ToValue(fmt.Sprintf("%v", v))
	}
	var parsed interface{}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return vm.ToValue(fmt.Sprintf("%v", v))
	}
	return vm.ToValue(parsed)
}

// jsonEqual compares two values for equality using JSON serialisation.
// This is deliberately simple: we don't need structural deep-equality — we
// just need to know if the script changed the value.
func jsonEqual(a, b interface{}) bool {
	aj, err1 := json.Marshal(a)
	bj, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aj) == string(bj)
}

