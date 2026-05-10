package config

import (
	"os"
	"strings"
)

type Config struct {
	// Server
	Port        string
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
	AllowStdioMCP     bool   // If true, allow starting local stdio MCP servers

	// Logging
	LogLevel string
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "8080"),
		CORSOrigins:       strings.Split(getEnv("CORS_ORIGINS", "http://localhost:5173"), ","),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://asm:asm_secret@localhost:5433/asm?sslmode=disable"),
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
		AdminUsername:     getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:     getEnv("ADMIN_PASSWORD", "pwd"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		LicensePublicKey:  getEnv("LICENSE_PUBLIC_KEY", ""),
		LicensePrivateKey: getEnv("LICENSE_PRIVATE_KEY", ""),
		AllowStdioMCP:     getEnv("PHAXA_ALLOW_STDIO_MCP", "false") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
