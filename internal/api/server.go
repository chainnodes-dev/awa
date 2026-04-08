package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/health"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/store"
)

type Server struct {
	httpServer *http.Server
	hub        *Hub
}

func NewServer(cfg *config.Config, engine *orchestrator.Engine, s store.Store, bus events.Bus, gen *designer.Generator, checker *health.Checker, sched SchedulerAdder, llmReg *llm.Registry) *Server {
	jwtSvc := auth.NewJWTService(cfg.JWTSecret)

	// Seed the default tenant and admin user on first boot.
	if cfg.AdminPassword != "" {
		go seedAdmin(cfg, s)
	}

	// Seed built-in demo workflows if the store is empty.
	go seedWorkflows(s)

	hub := NewHub(bus)
	hub.Start(context.Background())

	handlers := NewHandlers(engine, s, jwtSvc, gen, sched, llmReg)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(auth.RequestID())
	router.Use(MetricsMiddleware())

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	registerRoutes(router, handlers, hub, jwtSvc, checker)

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.Port),
			Handler: router,
		},
		hub: hub,
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// seedAdmin ensures the default tenant exists and creates the initial super_admin
// user on first boot when ADMIN_PASSWORD is set.
func seedAdmin(cfg *config.Config, s store.Store) {
	ctx := context.Background()

	// 1. Ensure the default tenant exists.
	defaultTenant, err := ensureDefaultTenant(ctx, s)
	if err != nil {
		slog.Error("Failed to ensure default tenant", "error", err)
		return
	}

	// 2. Create the admin user if no users exist yet.
	has, err := s.HasAnyUser(ctx)
	if err != nil || has {
		return
	}
	hash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		slog.Error("Failed to hash admin password", "error", err)
		return
	}
	u := &store.User{
		TenantID:     defaultTenant.ID,
		Username:     cfg.AdminUsername,
		PasswordHash: hash,
		Role:         string(auth.RoleSuperAdmin),
	}
	if err := s.CreateUser(ctx, u); err != nil {
		slog.Error("Failed to seed admin user", "error", err)
		return
	}
	slog.Info("Seeded super_admin user", "username", cfg.AdminUsername, "tenant", defaultTenant.Slug)
}

// ensureDefaultTenant creates the default tenant if it does not yet exist.
// Returns the default tenant (whether newly created or already present).
func ensureDefaultTenant(ctx context.Context, s store.Store) (*store.Tenant, error) {
	has, err := s.HasAnyTenant(ctx)
	if err != nil {
		return nil, err
	}
	if !has {
		t := &store.Tenant{
			ID:   store.DefaultTenantID,
			Name: "Default",
			Slug: "default",
		}
		if err := s.CreateTenant(ctx, t); err != nil {
			return nil, fmt.Errorf("create default tenant: %w", err)
		}
		slog.Info("Seeded default tenant")
		return t, nil
	}
	return s.GetTenant(ctx, store.DefaultTenantID)
}
