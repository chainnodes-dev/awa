package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/health"
)

func registerRoutes(r *gin.Engine, h *Handlers, hub *Hub, jwtSvc *auth.JWTService, checker *health.Checker) {
	// ── Observability (no auth required) ────────────────────────────────────
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// ── Health probes (no auth required) ────────────────────────────────────
	r.GET("/health",       checker.Live)  // backward-compat alias
	r.GET("/health/live",  checker.Live)
	r.GET("/health/ready", checker.Ready)


	// WebSocket — token validated inside ServeWS via query param.
	r.GET("/ws", func(c *gin.Context) {
		hub.ServeWS(c.Writer, c.Request, jwtSvc)
	})

	// Auth endpoints (no token required).
	authGroup := r.Group("/api/v1/auth")
	{
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.RefreshToken)
		authGroup.POST("/logout", auth.RequireAuth(jwtSvc), h.Logout)
	}

	// ── Protected ────────────────────────────────────────────────────────────
	api := r.Group("/api/v1", auth.RequireAuth(jwtSvc))
	{
		// Current user profile (all authenticated roles).
		api.GET("/users/me", h.GetMe)

		// User management (admin only).
		users := api.Group("/users", auth.RequireRole(auth.RoleAdmin))
		{
			users.GET("", h.ListUsers)
			users.POST("", h.CreateUser)
			users.PUT("/:id/role", h.UpdateUserRole)
			users.DELETE("/:id", h.DeleteUser)
		}

		// Workflow definitions.
		wf := api.Group("/workflows")
		{
			// Viewers may read; operators and admins may write.
			wf.GET("", h.ListWorkflows)
			wf.GET("/:name/versions", h.ListWorkflowVersions)
			wf.GET("/:name/v/:version", h.GetWorkflowByVersion)
			wf.GET("/:name/:version", h.GetWorkflow)
			wf.POST("", auth.RequireRole(auth.RoleAdmin, auth.RoleOperator), h.CreateWorkflow)
			wf.DELETE("/:name/:version", auth.RequireRole(auth.RoleAdmin), h.DeleteWorkflow)
		}

		// Workflow runs.
		runs := api.Group("/runs")
		{
			runs.GET("", h.ListRuns)
			runs.GET("/:id", h.GetRun)
			runs.GET("/:id/history", h.GetRunHistory)
			runs.POST("", auth.RequireRole(auth.RoleAdmin, auth.RoleOperator), h.StartRun)
			runs.POST("/:id/trigger", auth.RequireRole(auth.RoleAdmin, auth.RoleOperator), h.TriggerRun)
			runs.POST("/:id/signal", auth.RequireRole(auth.RoleAdmin, auth.RoleOperator), h.SignalHITL)
			runs.DELETE("/:id", auth.RequireRole(auth.RoleAdmin), h.DeleteRun)
		}

		// HITL management.
		api.GET("/hitl/pending", h.GetPendingHITL)

		// MCP Server Registry — readable by all authenticated users; write requires admin.
		api.GET("/mcp-servers", h.ListMCPServers)
		mcpServers := api.Group("/mcp-servers", auth.RequireRole(auth.RoleAdmin))
		{
			mcpServers.POST("", h.CreateMCPServer)
			mcpServers.POST("/discover", h.DiscoverMCPServer)
			mcpServers.PUT("/:id", h.UpdateMCPServer)
			mcpServers.DELETE("/:id", h.DeleteMCPServer)
			mcpServers.POST("/:id/ping", h.PingMCPServer)
		}

		// LLM provider configuration — admin only.
		llmCfg := api.Group("/llm-configs", auth.RequireRole(auth.RoleAdmin))
		{
			llmCfg.GET("", h.ListLLMConfigs)
			llmCfg.PUT("/:provider", h.UpsertLLMConfig)
			llmCfg.DELETE("/:provider", h.DeleteLLMConfig)
			llmCfg.POST("/default", h.SetDefaultProvider)
			llmCfg.POST("/:provider/test", h.TestLLMConnection)
		}

		// AI Workflow Designer — operator/admin only.
		designerGroup := api.Group("/designer", auth.RequireRole(auth.RoleAdmin, auth.RoleOperator))
		{
			designerGroup.POST("/generate", h.GenerateWorkflow)
			designerGroup.POST("/scaffold", h.ScaffoldHandler)
			designerGroup.POST("/scaffold-mcp", h.ScaffoldMCPHandler)
			designerGroup.POST("/codegen", h.CodegenHandler)
		}

		// Tenant management — super_admin only.
		tenants := api.Group("/tenants", auth.RequireRole(auth.RoleSuperAdmin))
		{
			tenants.POST("", h.CreateTenant)
			tenants.GET("", h.ListTenants)
			tenants.GET("/:id", h.GetTenant)
			tenants.DELETE("/:id", h.DeleteTenant)
		}
	}
}
