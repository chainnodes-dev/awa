package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asm-platform/asm/internal/catalog"
	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/store"
	"github.com/gin-gonic/gin"
)

func TestNodeCatalogEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		NodeCatalogSource: "../../configs/node-catalog.json",
	}
	h := &Handlers{
		store: memStore,
		cfg:   cfg,
	}

	t.Run("Get Built-in Catalog", func(t *testing.T) {
		entries, err := h.loadAndMergeNodeCatalog(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) == 0 {
			t.Fatal("expected at least 1 built-in entry")
		}
		found := false
		for _, e := range entries {
			if e.ID == "eu-vat-calculator" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected eu-vat-calculator in catalog")
		}
	})

	t.Run("Publish and Delete Custom Node", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/v1/node-catalog", h.GetNodeCatalog)
		r.POST("/api/v1/node-catalog", h.PublishNodeCatalog)
		r.DELETE("/api/v1/node-catalog/:id", h.DeleteNodeCatalog)

		// 1. Publish new node
		newNode := catalog.NodeCatalogEntry{
			ID:           "test-hasher",
			Name:         "Data Hasher",
			Version:      "1.0.0",
			Category:     "Security",
			CodeTemplate: "bb.hash = 'abc'; trigger('done');",
		}
		body, _ := json.Marshal(newNode)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/node-catalog", bytes.NewReader(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// 2. Fetch catalog and verify test-hasher is present
		req, _ = http.NewRequest(http.MethodGet, "/api/v1/node-catalog", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var list []catalog.NodeCatalogEntry
		_ = json.Unmarshal(w.Body.Bytes(), &list)
		hasNode := false
		for _, e := range list {
			if e.ID == "test-hasher" {
				hasNode = true
				break
			}
		}
		if !hasNode {
			t.Fatal("published node 'test-hasher' not found in GET response")
		}

		// 3. Delete node
		req, _ = http.NewRequest(http.MethodDelete, "/api/v1/node-catalog/test-hasher", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// 4. Verify deleted
		req, _ = http.NewRequest(http.MethodGet, "/api/v1/node-catalog", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		_ = json.Unmarshal(w.Body.Bytes(), &list)
		for _, e := range list {
			if e.ID == "test-hasher" {
				t.Fatal("expected 'test-hasher' to be deleted")
			}
		}
	})
}

