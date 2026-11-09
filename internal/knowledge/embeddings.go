package knowledge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/asm-platform/asm/internal/config"
)

// Embedder generates vector embeddings for given text snippets.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// CascadeEmbedder attempts embedding generation via configured providers
// with fallback to deterministic pseudo-vectors for offline/testing scenarios.
type CascadeEmbedder struct {
	client    *http.Client
	apiKey    string
	baseURL   string
	model     string
	ollamaURL string
	dim       int
}

// NewCascadeEmbedder initializes the embedder cascade using platform configuration.
func NewCascadeEmbedder(cfg *config.Config) *CascadeEmbedder {
	apiKey := cfg.EmbeddingAPIKey
	baseURL := cfg.EmbeddingBaseURL
	model := cfg.EmbeddingModel
	if model == "" {
		model = "text-embedding-3-small"
	}
	dim := cfg.EmbeddingDim
	if dim <= 0 {
		dim = 1536
	}

	// Auto-detect from OpenAI / OpenRouter if not explicitly set
	if apiKey == "" {
		if cfg.OpenAIKey != "" {
			apiKey = cfg.OpenAIKey
			if baseURL == "" {
				baseURL = "https://api.openai.com/v1"
			}
		} else if cfg.OpenRouterKey != "" {
			apiKey = cfg.OpenRouterKey
			if baseURL == "" {
				baseURL = "https://openrouter.ai/api/v1"
			}
			if cfg.EmbeddingModel == "" {
				model = "openai/text-embedding-3-small"
			}
		}
	} else if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &CascadeEmbedder{
		client: &http.Client{
			Timeout: 45 * time.Second,
		},
		apiKey:    apiKey,
		baseURL:   strings.TrimRight(baseURL, "/"),
		model:     model,
		ollamaURL: strings.TrimRight(cfg.OllamaURL, "/"),
		dim:       dim,
	}
}

// Embed generates an embedding vector for a single text string.
func (e *CascadeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	batch, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(batch) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return batch[0], nil
}

// EmbedBatch generates embeddings for a slice of texts.
func (e *CascadeEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// 1. Try OpenAI / OpenRouter compatible API if API key is configured
	if e.apiKey != "" && e.baseURL != "" {
		res, err := e.callOpenAIEmbedding(ctx, texts)
		if err == nil && len(res) == len(texts) {
			return res, nil
		}
		slog.Warn("OpenAI embedding failed, checking fallback", "error", err)
	}

	// 2. Try Ollama embeddings if endpoint is configured
	if e.ollamaURL != "" {
		res, err := e.callOllamaEmbedding(ctx, texts)
		if err == nil && len(res) == len(texts) {
			return res, nil
		}
		slog.Warn("Ollama embedding failed, falling back to deterministic pseudo-embeddings", "error", err)
	}

	// 3. Fallback: Deterministic semantic pseudo-vector (for unit tests / offline dev)
	slog.Info("Using deterministic pseudo-embeddings for offline RAG", "count", len(texts))
	results := make([][]float32, len(texts))
	for i, t := range texts {
		results[i] = generateDeterministicVector(t, e.dim)
	}
	return results, nil
}

type openAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (e *CascadeEmbedder) callOpenAIEmbedding(ctx context.Context, texts []string) ([][]float32, error) {
	endpoint := e.baseURL + "/embeddings"
	reqBody, _ := json.Marshal(openAIEmbeddingRequest{
		Model: e.model,
		Input: texts,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var res openAIEmbeddingResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, err
	}
	if res.Error != nil && res.Error.Message != "" {
		return nil, fmt.Errorf("embedding API error: %s", res.Error.Message)
	}

	out := make([][]float32, len(texts))
	for _, item := range res.Data {
		if item.Index >= 0 && item.Index < len(out) {
			out[item.Index] = normalizeVector(item.Embedding)
		}
	}
	return out, nil
}

type ollamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (e *CascadeEmbedder) callOllamaEmbedding(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		reqBody, _ := json.Marshal(ollamaEmbeddingRequest{
			Model:  "nomic-embed-text",
			Prompt: t,
		})
		endpoint := e.ollamaURL + "/api/embeddings"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("ollama HTTP %d", resp.StatusCode)
		}

		var res ollamaEmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, err
		}
		out[i] = padOrTruncate(normalizeVector(res.Embedding), e.dim)
	}
	return out, nil
}

func normalizeVector(v []float32) []float32 {
	var norm float64
	for _, val := range v {
		norm += float64(val) * float64(val)
	}
	if norm == 0 {
		return v
	}
	mag := float32(math.Sqrt(norm))
	out := make([]float32, len(v))
	for i, val := range v {
		out[i] = val / mag
	}
	return out
}

func padOrTruncate(v []float32, targetDim int) []float32 {
	if len(v) == targetDim {
		return v
	}
	out := make([]float32, targetDim)
	copy(out, v)
	return normalizeVector(out)
}

// generateDeterministicVector produces a repeatable, normalized pseudo-embedding
// based on token hashes so overlapping terms produce positive cosine similarity.
func generateDeterministicVector(text string, dim int) []float32 {
	vec := make([]float32, dim)
	h := sha256.New()

	words := strings.Fields(strings.ToLower(text))
	for _, word := range words {
		clean := strings.Trim(word, ".,!?:;\"'()[]{}")
		if len(clean) == 0 {
			continue
		}
		h.Reset()
		h.Write([]byte(clean))
		sum := h.Sum(nil)

		idx := int(binary.BigEndian.Uint32(sum[0:4])) % dim
		if idx < 0 {
			idx = -idx
		}
		vec[idx] += 1.0
	}

	return normalizeVector(vec)
}
