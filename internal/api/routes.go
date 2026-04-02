package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
)

func registerRoutes(r *gin.Engine, h *Handlers, hub *Hub, jwtSvc *auth.JWTService) {
	// ── Public ───────────────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

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
		}

		// HITL management.
		api.GET("/hitl/pending", h.GetPendingHITL)

		// MCP Server Registry — readable by all authenticated users.
		api.GET("/mcp-servers", h.ListMCPServers)

		// AI Workflow Designer — operator/admin only.
		designerGroup := api.Group("/designer", auth.RequireRole(auth.RoleAdmin, auth.RoleOperator))
		{
			designerGroup.POST("/generate", h.GenerateWorkflow)
		}
	}
}
