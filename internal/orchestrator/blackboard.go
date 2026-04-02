package orchestrator

import (
	"fmt"
	"sync"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// Blackboard is the thread-safe shared context for a single workflow run.
type Blackboard struct {
	mu     sync.RWMutex
	data   map[string]interface{}
	schema map[string]asmtypes.FieldDef
}

func NewBlackboard(schema map[string]asmtypes.FieldDef, initial map[string]interface{}) *Blackboard {
	bb := &Blackboard{
		data:   make(map[string]interface{}),
		schema: schema,
	}
	// Apply schema defaults
	for key, def := range schema {
		if def.Default != nil {
			bb.data[key] = def.Default
		}
	}
	// Apply initial values (overrides defaults)
	for k, v := range initial {
		bb.data[k] = v
	}
	return bb
}

// Read returns a copy of the current blackboard data.
func (b *Blackboard) Read() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()
	cp := make(map[string]interface{}, len(b.data))
	for k, v := range b.data {
		cp[k] = v
	}
	return cp
}

// Get returns a single field value.
func (b *Blackboard) Get(key string) (interface{}, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	v, ok := b.data[key]
	return v, ok
}

// Write applies a batch of updates, validating against the schema.
func (b *Blackboard) Write(updates map[string]interface{}) error {
	if err := b.validate(updates); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for k, v := range updates {
		b.data[k] = v
	}
	return nil
}

func (b *Blackboard) validate(updates map[string]interface{}) error {
	for key, value := range updates {
		def, ok := b.schema[key]
		if !ok {
			// Allow extra fields not in schema — agents may write metadata
			continue
		}
		if err := checkType(key, value, def.Type); err != nil {
			return err
		}
	}
	// Check required fields are present after applying updates
	merged := b.Read()
	for k, v := range updates {
		merged[k] = v
	}
	for key, def := range b.schema {
		if def.Required {
			if v, exists := merged[key]; !exists || v == nil {
				return fmt.Errorf("required blackboard field '%s' is missing", key)
			}
		}
	}
	return nil
}

func checkType(key string, value interface{}, expectedType string) error {
	if value == nil {
		return nil
	}
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' expects string, got %T", key, value)
		}
	case "number":
		switch value.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			// ok
		default:
			return fmt.Errorf("field '%s' expects number, got %T", key, value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' expects bool, got %T", key, value)
		}
	}
	return nil
}
