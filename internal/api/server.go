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
	"github.com/asm-platform/asm/internal/mcp"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/secrets"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/internal/triggers"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

type Server struct {
	httpServer *http.Server
	hub        *Hub
	mcpMgr     *mcp.Manager
	triggerMgr *triggers.Manager
	secretMgr  secrets.SecretManager
}

func NewServer(cfg *config.Config, engine *orchestrator.Engine, s store.Store, bus events.Bus, gen *designer.Generator, mcpEntries []designer.MCPServerEntry, checker *health.Checker, sched SchedulerAdder, llmReg *llm.Registry, secretMgr secrets.SecretManager) *Server {
	jwtSvc := auth.NewJWTService(cfg.JWTSecret)

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

	if pubKeyPEM == "" {
		pubKeyPEM = enterprise.ChainNodesPublicKey
		slog.Info("License verification enabled using built-in public key", "key_type", "RSA")
	} else {
		slog.Info("License verification enabled using configured public key", "key_type", "RSA")
	}

	pubKey, err := enterprise.ParseRSAPublicKey(pubKeyPEM)
	if err != nil {
		slog.Error("Failed to parse license public key", "error", err)
	} else {
		verifier = enterprise.NewVerifier(pubKey)
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

	mcpMgr := mcp.NewManager(s)
	triggerMgr := triggers.NewManager(s, func(ctx context.Context, tenantID, workflowName, version string, inputs map[string]interface{}) (*asmtypes.WorkflowRun, error) {
		return engine.StartRun(store.WithTenantID(ctx, tenantID), workflowName, version, inputs)
	}, secretMgr)

	handlers := NewHandlers(cfg, engine, s, jwtSvc, gen, mcpEntries, sched, llmReg, verifier, signer, mcpMgr, triggerMgr, secretMgr)

	router := gin.New()
	router.MaxMultipartMemory = 64 << 20 // 64 MiB
	router.Use(gin.Recovery())
	router.Use(auth.RequestID())
	router.Use(SecurityMiddleware(cfg.Environment, cfg.CORSOrigins))
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
		hub:        hub,
		mcpMgr:     mcpMgr,
		triggerMgr: triggerMgr,
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mcpMgr.Shutdown()
	return s.httpServer.Shutdown(ctx)
}

// SeedAdminWithTenant ensures the default tenant exists and creates the initial super_admin
// user on first boot when ADMIN_PASSWORD is set.
func SeedAdminWithTenant(cfg *config.Config, s store.Store, defaultTenant *store.Tenant) {
	ctx := context.Background()

	// 2. Create the admin user if no users exist yet.
	if cfg.AdminPassword == "" {
		return
	}
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

// EnsureDefaultTenant creates the default tenant if it does not yet exist.
// Returns the default tenant (whether newly created or already present).
func EnsureDefaultTenant(ctx context.Context, s store.Store) (*store.Tenant, error) {
	// 1. Try to find by the fixed ID first.
	if t, err := s.GetTenant(ctx, store.DefaultTenantID); err == nil {
		return t, nil
	}

	// 2. Try to find by the "default" slug.
	if t, err := s.GetTenantBySlug(ctx, "default"); err == nil {
		return t, nil
	}

	// 3. If the fixed ID doesn't exist, we MUST create it or ensure a viable default.
	// We want to avoid foreign key violations for handlers that rely on DefaultTenantID.
	t := &store.Tenant{
		ID:   store.DefaultTenantID,
		Name: "Chain Nodes",
		Slug: "default",
	}
	if err := s.CreateTenant(ctx, t); err != nil {
		// If it already exists (e.g. race condition), just return it.
		if t2, err2 := s.GetTenant(ctx, store.DefaultTenantID); err2 == nil {
			return t2, nil
		}
		// If we can't create it and it doesn't exist, fallback to whatever we have.
		tenants, _ := s.ListTenants(ctx)
		if len(tenants) > 0 {
			return tenants[0], nil
		}
		return nil, fmt.Errorf("failed to create default tenant: %w", err)
	}
	slog.Info("Ensured canonical default tenant exists", "id", t.ID)
	return t, nil
}


// SeedLLMConfigs ensured that at least one LLM provider is configured
// by reading from environment variables on first boot.
func SeedLLMConfigs(cfg *config.Config, s store.Store, tenantID string) {
	ctx := store.WithTenantID(context.Background(), tenantID)

	hasAny, err := s.HasAnyLLMConfig(ctx)
	if err != nil || hasAny {
		return
	}

	type seed struct{ provider, apiKey, baseURL, model string }
	seeds := []seed{
		{"anthropic", cfg.AnthropicKey, "", "claude-3-5-sonnet-latest"},
		{"openai", cfg.OpenAIKey, "", "gpt-4o"},
		{"grok", cfg.GrokKey, "", "grok-3"},
		{"deepseek", cfg.DeepseekKey, "", "deepseek-v4-flash"},
		{"gemini", cfg.GeminiKey, "", "gemini-2.0-flash"},
		{"ollama", "", cfg.OllamaURL, cfg.OllamaModel},
	}

	for _, sd := range seeds {
		if sd.provider == "ollama" {
			if sd.baseURL == "" {
				continue
			}
		} else if sd.apiKey == "" {
			continue
		}

		err := s.UpsertLLMConfig(ctx, &store.LLMConfig{
			TenantID:     tenantID,
			Provider:     sd.provider,
			APIKey:       sd.apiKey,
			BaseURL:      sd.baseURL,
			DefaultModel: sd.model,
			Enabled:      true,
		})
		if err != nil {
			slog.Error("Failed to seed LLM config", "provider", sd.provider, "error", err)
		} else {
			slog.Info("Seeded LLM config", "provider", sd.provider)
		}
	}

	if cfg.LLMProvider != "" {
		_ = s.SetDefaultProvider(ctx, cfg.LLMProvider)
	}
}
