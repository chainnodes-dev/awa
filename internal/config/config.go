package config

import (
	"os"
	"strings"
)

type Config struct {
	// Server
	Port        string
	Environment string   // "development" | "production"
	CORSOrigins []string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Temporal
	TemporalAddress   string
	TemporalNamespace string
	TemporalTaskQueue string

	// LLM
	LLMProvider  string // "anthropic" | "openai" | "ollama"
	AnthropicKey string
	OpenAIKey    string
	DefaultModel string
	// Ollama (local LLM server — no API key required)
	OllamaURL   string // e.g. "http://mintbox:11434"
	OllamaModel string // e.g. "llama3.2"
	// OpenAI-compatible cloud providers
	GrokKey     string // xAI Grok
	DeepseekKey string // Deepseek
	GeminiKey   string // Google AI Studio

	// Auth
	JWTSecret     string // HS256 signing key — must be set in production
	AdminUsername string // seed admin username on first boot
	AdminPassword    string // seed admin password on first boot
	LicensePublicKey  string // PEM-encoded RSA public key for verification
	LicensePrivateKey string // PEM-encoded RSA private key for signing (optional)
	MCPMarketSource   string // URL or local file path for MCP Marketplace data

	// Logging
	LogLevel string

	// Telemetry
	EnableTelemetry bool
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "8080"),
		Environment:       getEnv("ENVIRONMENT", "development"),
		CORSOrigins:       strings.Split(getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:5174"), ","),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://chainnode:chainnode_secret@localhost:5433/chainnode?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379"),
		TemporalAddress:   getEnv("TEMPORAL_ADDRESS", "localhost:7233"),
		TemporalNamespace: getEnv("TEMPORAL_NAMESPACE", "default"),
		TemporalTaskQueue: getEnv("TEMPORAL_TASK_QUEUE", "phaxa-workers"),
		LLMProvider:       getEnv("LLM_PROVIDER", ""),
		AnthropicKey:      getEnv("ANTHROPIC_API_KEY", ""),
		OpenAIKey:         getEnv("OPENAI_API_KEY", ""),
		DefaultModel:      getEnv("DEFAULT_MODEL", "claude-sonnet-4-6"),
		OllamaURL:         getEnv("OLLAMA_URL", "http://mintbox:11434"),
		OllamaModel:       getEnv("OLLAMA_MODEL", "llama3.2"),
		GrokKey:           getEnv("GROK_API_KEY", ""),
		DeepseekKey:       getEnv("DEEPSEEK_API_KEY", ""),
		GeminiKey:         getEnv("GEMINI_API_KEY", ""),
		JWTSecret:         getEnv("JWT_SECRET", "dev-insecure-secret-change-in-prod"),
		AdminUsername:     getEnv("ADMIN_USERNAME", ""),
		AdminPassword:     getEnv("ADMIN_PASSWORD", ""),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		LicensePublicKey:  getEnv("LICENSE_PUBLIC_KEY", ""),
		LicensePrivateKey: getEnv("LICENSE_PRIVATE_KEY", ""),
		MCPMarketSource:   getEnv("MCP_MARKET_SOURCE", "mcp_registry.yaml"),
		EnableTelemetry:   getEnv("ENABLE_TELEMETRY", "false") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
