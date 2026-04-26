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
	"github.com/asm-platform/asm/internal/enterprise"
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

	// Seed LLM configs from env vars if not present.
	go seedLLMConfigs(cfg, s)

	hub := NewHub(bus)
	hub.Start(context.Background())

	var verifier *enterprise.Verifier
	var signer *enterprise.Signer

	// Load keys from DB if present, otherwise fallback to config.
	pubKeyPEM, err := s.GetSystemSetting(context.Background(), "license_public_key")
	if err != nil || pubKeyPEM == "" {
		pubKeyPEM = cfg.LicensePublicKey
	}

	privKeyPEM, err := s.GetSystemSetting(context.Background(), "license_private_key")
	if err != nil || privKeyPEM == "" {
		privKeyPEM = cfg.LicensePrivateKey
	}

	if pubKeyPEM != "" {
		pubKey, err := enterprise.ParseRSAPublicKey(pubKeyPEM)
		if err != nil {
			slog.Error("Failed to parse license public key", "error", err)
		} else {
			verifier = enterprise.NewVerifier(pubKey)
			slog.Info("License verification enabled", "key_type", "RSA")
		}
	} else {
		slog.Warn("License verification disabled: no public key found in DB or env")
	}

	if privKeyPEM != "" {
		privKey, err := enterprise.ParseRSAPrivateKey(privKeyPEM)
		if err != nil {
			slog.Error("Failed to parse license private key", "error", err)
		} else {
			signer = enterprise.NewSigner(privKey)
			slog.Info("License signing enabled")
		}
	}

	handlers := NewHandlers(engine, s, jwtSvc, gen, sched, llmReg, verifier, signer)

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
	// 1. Try to find by the fixed ID first.
	if t, err := s.GetTenant(ctx, store.DefaultTenantID); err == nil {
		return t, nil
	}

	// 2. Try to find by the "default" slug.
	if t, err := s.GetTenantBySlug(ctx, "default"); err == nil {
		return t, nil
	}

	// 3. If neither exists, check if ANY tenant exists.
	// If the DB is completely empty, create the canonical default tenant.
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
		slog.Info("Seeded default tenant", "id", t.ID)
		return t, nil
	}

	// 4. If some other tenant exists but no "default" one, return the first one
	// found so that seeding can at least attach the admin user to something.
	tenants, err := s.ListTenants(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing tenants: %w", err)
	}
	if len(tenants) == 0 {
		return nil, fmt.Errorf("HasAnyTenant was true but ListTenants returned 0 rows")
	}
	return tenants[0], nil
}


// seedLLMConfigs ensures that at least one LLM provider is configured
// by reading from environment variables on first boot.
func seedLLMConfigs(cfg *config.Config, s store.Store) {
	ctx := store.WithTenantID(context.Background(), store.DefaultTenantID)

	// Only seed if no LLM configs exist for the default tenant.
	existing, err := s.ListLLMConfigs(ctx)
	if err != nil || len(existing) > 0 {
		return
	}

	// 1. Seed Ollama if OLLAMA_URL is set.
	if cfg.OllamaURL != "" {
		err := s.UpsertLLMConfig(ctx, &store.LLMConfig{
			TenantID:     store.DefaultTenantID,
			Provider:     "ollama",
			BaseURL:      cfg.OllamaURL,
			DefaultModel: cfg.OllamaModel,
			Enabled:      true,
		})
		if err != nil {
			slog.Error("Failed to seed Ollama config", "error", err)
		} else {
			slog.Info("Seeded Ollama LLM config", "url", cfg.OllamaURL)
		}
	}

	// 2. Set the default provider if LLM_PROVIDER is set.
	if cfg.LLMProvider != "" {
		if err := s.SetDefaultProvider(ctx, cfg.LLMProvider); err != nil {
			slog.Error("Failed to set default LLM provider", "provider", cfg.LLMProvider, "error", err)
		} else {
			slog.Info("Set default LLM provider", "provider", cfg.LLMProvider)
		}
	}
}
