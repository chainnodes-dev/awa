package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

type Handlers struct {
	engine    *orchestrator.Engine
	store     store.Store
	jwtSvc    *auth.JWTService
	// generator is nil when no LLM key is configured.
	generator *designer.Generator
}

func NewHandlers(engine *orchestrator.Engine, s store.Store, jwtSvc *auth.JWTService, gen *designer.Generator) *Handlers {
	return &Handlers{engine: engine, store: s, jwtSvc: jwtSvc, generator: gen}
}

// -- Workflow Definitions --

func (h *Handlers) ListWorkflows(c *gin.Context) {
	defs, err := h.store.ListDefinitions(c.Request.Context())
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

	if err := h.store.SaveDefinition(c.Request.Context(), def, yamlSource); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, def)
}

func (h *Handlers) DeleteWorkflow(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")
	if err := h.store.DeleteDefinition(c.Request.Context(), name, version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// -- Workflow Runs --

func (h *Handlers) ListRuns(c *gin.Context) {
	filter := store.RunFilter{
		WorkflowName: c.Query("workflow"),
		Status:       asmtypes.RunStatus(c.Query("status")),
		Limit:        50,
	}
	runs, err := h.engine.ListRuns(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, runs)
}

func (h *Handlers) GetRun(c *gin.Context) {
	run, err := h.engine.GetRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, run)
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
