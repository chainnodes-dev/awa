package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/store"
)

type Server struct {
	httpServer *http.Server
	hub        *Hub
}

func NewServer(cfg *config.Config, engine *orchestrator.Engine, s store.Store, bus events.Bus, gen *designer.Generator) *Server {
	jwtSvc := auth.NewJWTService(cfg.JWTSecret)

	// Seed the default admin user if no users exist yet.
	if cfg.AdminPassword != "" {
		go seedAdmin(cfg, s, jwtSvc)
	}

	// Seed built-in demo workflows if the store is empty.
	go seedWorkflows(s)

	hub := NewHub(bus)
	hub.Start(context.Background())

	handlers := NewHandlers(engine, s, jwtSvc, gen)

	router := gin.Default()

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	registerRoutes(router, handlers, hub, jwtSvc)

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

// seedAdmin creates the default admin user on first boot when ADMIN_PASSWORD is set.
func seedAdmin(cfg *config.Config, s store.Store, _ *auth.JWTService) {
	ctx := context.Background()
	has, err := s.HasAnyUser(ctx)
	if err != nil || has {
		return
	}
	hash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		log.Printf("[server] failed to hash admin password: %v", err)
		return
	}
	u := &store.User{
		Username:     cfg.AdminUsername,
		PasswordHash: hash,
		Role:         string(auth.RoleAdmin),
	}
	if err := s.CreateUser(ctx, u); err != nil {
		log.Printf("[server] failed to seed admin user: %v", err)
		return
	}
	log.Printf("[server] seeded admin user '%s'", cfg.AdminUsername)
}
