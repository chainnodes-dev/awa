package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/enterprise"
	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// SchedulerAdder is the minimal scheduler interface needed by the handlers.
// Using an interface keeps the api package free of a direct import on internal/scheduler.
type SchedulerAdder interface {
	AddWorkflow(def *asmtypes.WorkflowDef) error
	RemoveWorkflow(name, version string)
}

type Handlers struct {
	engine      *orchestrator.Engine
	store       store.Store
	jwtSvc      *auth.JWTService
	// generator is nil when no LLM key is configured.
	generator   *designer.Generator
	scheduler       SchedulerAdder
	llmRegistry     *llm.Registry
	licenseVerifier *enterprise.Verifier
}

func NewHandlers(engine *orchestrator.Engine, s store.Store, jwtSvc *auth.JWTService, gen *designer.Generator, sched SchedulerAdder, llmReg *llm.Registry, verifier *enterprise.Verifier) *Handlers {
	return &Handlers{
		engine:          engine,
		store:           s,
		jwtSvc:          jwtSvc,
		generator:       gen,
		scheduler:       sched,
		llmRegistry:     llmReg,
		licenseVerifier: verifier,
	}
}

// -- Workflow Definitions --

func (h *Handlers) ListWorkflows(c *gin.Context) {
	filter := store.DefinitionFilter{
		Limit:        queryInt(c, "limit", 50),
		Offset:       queryInt(c, "offset", 0),
		ReusableOnly: c.Query("reusable") == "true",
	}
	defs, err := h.store.ListDefinitions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, defs)
}

func (h *Handlers) GetWorkflow(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")
	def, yaml, err := h.store.GetDefinition(c.Request.Context(), name, version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"definition": def, "yaml": yaml})
}

func (h *Handlers) CreateWorkflow(c *gin.Context) {
	var body struct {
		YAML string `json:"yaml" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	def, yamlSource, err := asmtypes.LoadFromYAML([]byte(body.YAML))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.checkCreateWorkflowLimit(c); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.SaveDefinition(c.Request.Context(), def, yamlSource); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Register (or update) cron schedule if the workflow carries one.
	if h.scheduler != nil {
		if err := h.scheduler.AddWorkflow(def); err != nil {
			slog.Warn("Failed to register workflow schedule", "workflow", def.Metadata.Name, "error", err)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"definition":     def,
		"version_number": def.Metadata.VersionNumber,
	})
}

func (h *Handlers) ListWorkflowVersions(c *gin.Context) {
	name := c.Param("name")
	versions, err := h.store.ListDefinitionVersions(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, versions)
}

func (h *Handlers) GetWorkflowByVersion(c *gin.Context) {
	name := c.Param("name")
	vnum, err := strconv.Atoi(c.Param("version"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version must be an integer"})
		return
	}
	def, yaml, err := h.store.GetDefinitionByVersion(c.Request.Context(), name, vnum)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"definition": def, "yaml": yaml})
}

func (h *Handlers) DeleteWorkflow(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")
	if err := h.store.DeleteDefinition(c.Request.Context(), name, version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Deregister any cron schedule for this workflow.
	if h.scheduler != nil {
		h.scheduler.RemoveWorkflow(name, version)
	}
	c.Status(http.StatusNoContent)
}

// -- Workflow Runs --

func (h *Handlers) ListRuns(c *gin.Context) {
	filter := store.RunFilter{
		TenantID:     store.TenantIDFromContext(c.Request.Context()),
		WorkflowName: c.Query("workflow"),
		Status:       asmtypes.RunStatus(c.Query("status")),
		CurrentState: c.Query("state"),
		Limit:        queryInt(c, "limit", 50),
		Offset:       queryInt(c, "offset", 0),
	}
	if from := c.Query("started_from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.StartedFrom = &t
		}
	}
	if to := c.Query("started_to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.StartedTo = &t
		}
	}
	runs, err := h.engine.ListRuns(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, runs)
}

// queryInt reads a query parameter as an integer, returning defaultVal if absent or invalid.
func queryInt(c *gin.Context, key string, defaultVal int) int {
	if s := c.Query(key); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			return n
		}
	}
	return defaultVal
}

func (h *Handlers) GetRun(c *gin.Context) {
	run, err := h.engine.GetRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, run)
}

func (h *Handlers) DeleteRun(c *gin.Context) {
	if err := h.engine.DeleteRun(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) TerminateRun(c *gin.Context) {
	if err := h.engine.TerminateRun(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) GetRunHistory(c *gin.Context) {
	transitions, err := h.store.ListTransitions(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transitions)
}

func (h *Handlers) StartRun(c *gin.Context) {
	var body struct {
		WorkflowName    string                 `json:"workflow_name" binding:"required"`
		WorkflowVersion string                 `json:"workflow_version" binding:"required"`
		Input           map[string]interface{} `json:"input"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.checkRunLimit(c); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
		return
	}

	run, err := h.engine.StartRun(c.Request.Context(), body.WorkflowName, body.WorkflowVersion, body.Input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, run)
}

func (h *Handlers) TriggerRun(c *gin.Context) {
	var req asmtypes.TriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.RunID = c.Param("id")

	if err := h.engine.Trigger(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handlers) SignalHITL(c *gin.Context) {
	var sig asmtypes.HITLSignal
	if err := c.ShouldBindJSON(&sig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sig.RunID = c.Param("id")

	if err := h.engine.SignalHITL(c.Request.Context(), sig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handlers) GetPendingHITL(c *gin.Context) {
	requests, err := h.store.ListPendingHITL(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, requests)
}

func (h *Handlers) GetMCPLogs(c *gin.Context) {
	runID := c.Param("id")
	logs, err := h.store.ListMCPCalls(c.Request.Context(), runID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}
// -- License Gating Helpers --

func (h *Handlers) getLicenseClaims(ctx context.Context) (*enterprise.LicenseClaims, error) {
	tenantID := store.TenantIDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	tenant, err := h.store.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant.LicenseToken == "" {
		return &enterprise.LicenseClaims{Tier: enterprise.TierFree, MaxWorkflows: 5, MaxRuns: 100}, nil
	}
	return h.licenseVerifier.Verify(tenant.LicenseToken)
}

func (h *Handlers) checkCreateWorkflowLimit(c *gin.Context) error {
	claims, err := h.getLicenseClaims(c.Request.Context())
	if err != nil {
		return err
	}
	if claims.MaxWorkflows == 0 { // Unlimited
		return nil
	}

	count, err := h.store.CountDefinitions(c.Request.Context())
	if err != nil {
		return err
	}
	if count >= claims.MaxWorkflows {
		return fmt.Errorf("tier limit reached: maximum %d workflows allowed for your tier", claims.MaxWorkflows)
	}
	return nil
}

func (h *Handlers) checkRunLimit(c *gin.Context) error {
	claims, err := h.getLicenseClaims(c.Request.Context())
	if err != nil {
		return err
	}
	if claims.MaxRuns == 0 { // Unlimited
		return nil
	}

	now := time.Now()
	beginningOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	filter := store.RunFilter{
		TenantID:    store.TenantIDFromContext(c.Request.Context()),
		StartedFrom: &beginningOfMonth,
	}
	count, err := h.store.CountRuns(c.Request.Context(), filter)
	if err != nil {
		return err
	}
	if count >= claims.MaxRuns {
		return fmt.Errorf("tier limit reached: maximum %d runs per month allowed for your tier", claims.MaxRuns)
	}
	return nil
}
