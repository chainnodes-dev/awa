package config

import (
	"os"
	"strconv"
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
	LLMProvider   string // "openrouter" | "anthropic" | "openai" | "ollama"
	OpenRouterKey string
	AnthropicKey  string
	OpenAIKey     string
	DefaultModel string
	// Ollama (local LLM server — no API key required)
	OllamaURL   string // e.g. "http://mintbox:11434"
	OllamaModel string // e.g. "llama3.2"
	// OpenAI-compatible cloud providers
	GrokKey     string // xAI Grok
	DeepseekKey string // Deepseek
	GeminiKey   string // Google AI Studio

	// Embeddings & Vector RAG
	EmbeddingAPIKey  string
	EmbeddingBaseURL string
	EmbeddingModel   string
	EmbeddingDim     int

	// Inbound Email & IMAP
	IMAPHost         string
	IMAPPort         int
	IMAPUsername     string
	IMAPPassword     string
	IMAPMailbox      string
	IMAPUseTLS       bool
	IMAPPollInterval string // e.g. "60s"

	// Auth
	JWTSecret     string // HS256 signing key — must be set in production
	AdminUsername string // seed admin username on first boot
	AdminPassword    string // seed admin password on first boot
	LicensePublicKey  string // PEM-encoded RSA public key for verification
	LicensePrivateKey string // PEM-encoded RSA private key for signing (optional)
	MCPMarketSource   string // URL or local file path for MCP Marketplace data
	NodeCatalogSource string // URL or local file path for custom node catalog data

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
		TemporalTaskQueue: getEnv("TEMPORAL_TASK_QUEUE", "chainnodes-workers"),
		LLMProvider:       getEnv("LLM_PROVIDER", ""),
		OpenRouterKey:     getEnv("OPENROUTER_API_KEY", ""),
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
		MCPMarketSource:   getEnv("MCP_MARKET_SOURCE", "configs/mcp-market.json"),
		NodeCatalogSource: getEnv("NODE_CATALOG_SOURCE", "configs/node-catalog.json"),
		EnableTelemetry:   getEnv("ENABLE_TELEMETRY", "false") == "true",
		EmbeddingAPIKey:   getEnv("EMBEDDING_API_KEY", ""),
		EmbeddingBaseURL:  getEnv("EMBEDDING_BASE_URL", ""),
		EmbeddingModel:    getEnv("EMBEDDING_MODEL", "text-embedding-3-small"),
		EmbeddingDim:      getEnvInt("EMBEDDING_DIM", 1536),
		IMAPHost:         getEnv("IMAP_HOST", ""),
		IMAPPort:         getEnvInt("IMAP_PORT", 993),
		IMAPUsername:     getEnv("IMAP_USERNAME", ""),
		IMAPPassword:     getEnv("IMAP_PASSWORD", ""),
		IMAPMailbox:      getEnv("IMAP_MAILBOX", "INBOX"),
		IMAPUseTLS:       getEnv("IMAP_USE_TLS", "true") == "true",
		IMAPPollInterval: getEnv("IMAP_POLL_INTERVAL", "60s"),
	}
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
