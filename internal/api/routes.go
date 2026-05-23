package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/health"
)

func registerRoutes(r *gin.Engine, h *Handlers, hub *Hub, jwtSvc *auth.JWTService, checker *health.Checker) {
	// ── Observability & Health (No Auth) ────────────────────────────────────
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/health", checker.Live)
	r.GET("/health/live", checker.Live)
	r.GET("/health/ready", checker.Ready)

	// WebSocket
	r.GET("/ws", func(c *gin.Context) {
		hub.ServeWS(c.Writer, c.Request, jwtSvc)
	})

	// ── API V1 ──────────────────────────────────────────────────────────────
	v1 := r.Group("/api/v1")
	{
		// 1. PUBLIC AUTH (No Auth)
		// We explicitly define these BEFORE the protected group to avoid shadowing.
		authGroup := v1.Group("/auth")
		{
			authGroup.GET("/status", h.GetAuthStatus)
			authGroup.POST("/setup", h.SetupAdmin)
			authGroup.POST("/login", h.Login)
			authGroup.POST("/refresh", h.RefreshToken)
			authGroup.GET("/oidc/login", h.OIDCLogin)
			authGroup.GET("/oidc/callback", h.OIDCCallback)
		}

		// Webhook Triggers
		v1.POST("/triggers/webhook/:tenant_id/:workflow_name/:version/:trigger_name", h.HandleWebhook)

		// 2. PROTECTED API (Requires JWT)
		protected := v1.Group("", auth.RequireAuth(jwtSvc))
		{
			protected.POST("/auth/logout", h.Logout)
			protected.GET("/users/me", h.GetMe)

			// Admin-only users
			users := protected.Group("/users", auth.RequireRole(auth.RoleAdmin))
			{
				users.GET("", h.ListUsers)
				users.POST("", h.CreateUser)
				users.PUT("/:id/role", h.UpdateUserRole)
				users.DELETE("/:id", h.DeleteUser)
			}

			// Workflows
			wf := protected.Group("/workflows")
			{
				wf.GET("", h.ListWorkflows)
				wf.GET("/:name/versions", h.ListWorkflowVersions)
				wf.GET("/:name/v/:version", h.GetWorkflowByVersion)
				wf.GET("/:name/:version", h.GetWorkflow)
				wf.POST("", auth.RequireRole(auth.RoleAdmin, auth.RoleEditor, auth.RoleOperator), h.CreateWorkflow)
				wf.DELETE("/:name/:version", auth.RequireRole(auth.RoleAdmin, auth.RoleEditor), h.DeleteWorkflow)
			}

			// Runs
			runs := protected.Group("/runs")
			{
				runs.GET("", h.ListRuns)
				runs.GET("/:id", h.GetRun)
				runs.GET("/:id/history", h.GetRunHistory)
				runs.GET("/:id/mcp-logs", h.GetMCPLogs)
				runs.POST("", auth.RequireRole(auth.RoleAdmin, auth.RoleRunner, auth.RoleOperator), h.StartRun)
				runs.POST("/:id/trigger", auth.RequireRole(auth.RoleAdmin, auth.RoleRunner, auth.RoleOperator), h.TriggerRun)
				runs.POST("/:id/signal", auth.RequireRole(auth.RoleAdmin, auth.RoleRunner, auth.RoleOperator), h.SignalHITL)
				runs.POST("/:id/chat", auth.RequireRole(auth.RoleAdmin, auth.RoleRunner, auth.RoleOperator), h.SendChat)
				runs.POST("/:id/terminate", auth.RequireRole(auth.RoleAdmin, auth.RoleRunner, auth.RoleOperator), h.TerminateRun)
				runs.DELETE("/:id", auth.RequireRole(auth.RoleAdmin), h.DeleteRun)
			}

			protected.GET("/hitl/pending", h.GetPendingHITL)
			protected.GET("/mcp-market", h.GetMCPMarket)
			protected.GET("/mcp-servers", h.ListMCPServers)

			mcpServers := protected.Group("/mcp-servers", auth.RequireRole(auth.RoleAdmin, auth.RoleEditor))
			{
				mcpServers.POST("", h.CreateMCPServer)
				mcpServers.POST("/discover", h.DiscoverMCPServer)
				mcpServers.PUT("/:id", h.UpdateMCPServer)
				mcpServers.DELETE("/:id", h.DeleteMCPServer)
				mcpServers.POST("/:id/ping", h.PingMCPServer)
			}

			llmCfg := protected.Group("/llm-configs", auth.RequireRole(auth.RoleAdmin, auth.RoleEditor))
			{
				llmCfg.GET("", h.ListLLMConfigs)
				llmCfg.PUT("/:provider", h.UpsertLLMConfig)
				llmCfg.DELETE("/:provider", h.DeleteLLMConfig)
				llmCfg.POST("/default", h.SetDefaultProvider)
				llmCfg.POST("/:provider/test", h.TestLLMConnection)
			}

			designerGroup := protected.Group("/designer", auth.RequireRole(auth.RoleAdmin, auth.RoleEditor, auth.RoleOperator))
			{
				designerGroup.POST("/generate", h.GenerateWorkflow)
				designerGroup.POST("/scaffold", h.ScaffoldHandler)
				designerGroup.POST("/scaffold-mcp", h.ScaffoldMCPHandler)
				designerGroup.POST("/codegen", h.CodegenHandler)
				designerGroup.GET("/prompts", h.GetPrompts)
				designerGroup.PUT("/prompts/:id", h.UpdatePrompt)
				designerGroup.POST("/debug", h.DebugWorkflowHandler)
				designerGroup.POST("/process/analyse", h.ProcessAnalyseHandler)
				designerGroup.POST("/process/summarise", h.ProcessSummariseHandler)

				pipeline := designerGroup.Group("/pipeline")
				{
					pipeline.POST("/decompose", h.PipelineDecomposeHandler)
					pipeline.POST("/categorise", h.PipelineCategoriseHandler)
					pipeline.POST("/wire", h.PipelineWireHandler)
					pipeline.POST("/implement", h.PipelineImplementHandler)
				}
			}

			apiKeys := protected.Group("/api-keys", auth.RequireRole(auth.RoleAdmin))
			{
				apiKeys.POST("", h.CreateAPIKey)
				apiKeys.GET("", h.ListAPIKeys)
				apiKeys.DELETE("/:id", h.RevokeAPIKey)
			}

			ent := protected.Group("/enterprise")
			{
				ent.GET("/branding", h.GetBranding)
				entAdmin := ent.Group("", auth.RequireRole(auth.RoleAdmin))
				{
					entAdmin.POST("/license", h.SetLicense)
					entAdmin.GET("/status", h.GetLicenseStatus)
					entAdmin.POST("/branding", h.UpdateBranding)
					entAdmin.GET("/secrets", h.GetSecrets)
					entAdmin.POST("/secrets", h.UpdateSecrets)
					entAdmin.GET("/audit-logs", h.ListAuditLogs)
					entAdmin.GET("/analytics", h.GetAnalytics)
					entAdmin.PUT("/oidc", h.SetIDPConfig)
					entAdmin.GET("/mcp-market-source", h.GetMarketSource)
					entAdmin.PUT("/mcp-market-source", h.UpdateMarketSource)
				}
			}

			admin := protected.Group("/admin", auth.RequireRole(auth.RoleAdmin, auth.RoleSuperAdmin))
			{
				admin.GET("/telemetry", h.GetTelemetryStatus)
				admin.POST("/telemetry/toggle", h.ToggleTelemetry)
			}
		}

		// 3. MIXED AUTH (JWT or API Key)
		mixed := v1.Group("", RequireAuthOrAPIKey(jwtSvc, h.store))
		{
			mixed.POST("/uploads", h.UploadFile)
			mixed.GET("/uploads/:id/:filename", h.GetUploadedFile)
			mixed.POST("/invoke/:name", h.InvokeProcess)
			mixed.GET("/invoke/:name/runs/:run_id", h.GetProcessInvocation)
		}
	}
}
