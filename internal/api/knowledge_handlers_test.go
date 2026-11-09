package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/knowledge"
	"github.com/asm-platform/asm/internal/store"
	"github.com/gin-gonic/gin"
)

func setupTestKnowledgeHandler(t *testing.T) (*Handlers, *gin.Engine, string) {
	gin.SetMode(gin.TestMode)
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		EmbeddingModel: "text-embedding-3-small",
		EmbeddingDim:   64,
	}

	jwtSvc := auth.NewJWTService("test-secret-must-be-at-least-32-bytes-long")
	tenantID := store.DefaultTenantID

	h := &Handlers{
		store:  memStore,
		cfg:    cfg,
		jwtSvc: jwtSvc,
	}
	h.SetKnowledgeService(knowledge.NewService(memStore, cfg))

	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		kGroup := v1.Group("/knowledge")
		{
			kGroup.POST("/upload", h.UploadKnowledgeDocument)
			kGroup.GET("/documents", h.ListKnowledgeDocuments)
			kGroup.DELETE("/documents/:id", h.DeleteKnowledgeDocument)
			kGroup.GET("/collections", h.ListKnowledgeCollections)
			kGroup.POST("/search", h.SearchKnowledge)
		}
	}

	token, err := jwtSvc.GenerateAccessToken("user-1", tenantID, "admin", "admin")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return h, router, token
}

func TestKnowledgeAPI_UploadAndSearch(t *testing.T) {
	h, router, token := setupTestKnowledgeHandler(t)
	defer h.GetKnowledgeService().Close()

	// 1. Upload via multipart
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "IT_Security_Policy.txt")
	part.Write([]byte("Password requirements: Minimum 16 characters with symbols. MFA is mandatory for all production systems."))
	writer.WriteField("collection", "security")
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/knowledge/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var createdDoc store.KnowledgeDocument
	if err := json.Unmarshal(w.Body.Bytes(), &createdDoc); err != nil {
		t.Fatalf("unmarshal created doc failed: %v", err)
	}
	if createdDoc.Collection != "security" {
		t.Errorf("expected collection security, got %s", createdDoc.Collection)
	}
	if createdDoc.Filename != "IT_Security_Policy.txt" {
		t.Errorf("expected filename IT_Security_Policy.txt, got %s", createdDoc.Filename)
	}

	// 2. Upload via JSON Base64
	jsonBody, _ := json.Marshal(map[string]string{
		"name":           "Code_Of_Conduct.txt",
		"collection":     "compliance",
		"content_base64": base64.StdEncoding.EncodeToString([]byte("Zero tolerance for harassment or discriminatory behavior.")),
		"content_type":   "text/plain",
	})
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/knowledge/upload", bytes.NewReader(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w2.Code, w2.Body.String())
	}

	// 3. Wait for background worker to index
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		doc, _ := h.store.GetDocument(req.Context(), createdDoc.ID)
		if doc != nil && doc.Status == "ready" {
			break
		}
	}

	// 4. List collections
	reqColls, _ := http.NewRequest(http.MethodGet, "/api/v1/knowledge/collections", nil)
	reqColls.Header.Set("Authorization", "Bearer "+token)
	wColls := httptest.NewRecorder()
	router.ServeHTTP(wColls, reqColls)
	if wColls.Code != http.StatusOK {
		t.Fatalf("list collections failed: %d", wColls.Code)
	}
	var colls []string
	_ = json.Unmarshal(wColls.Body.Bytes(), &colls)
	if len(colls) < 2 {
		t.Errorf("expected at least 2 collections, got %d (%v)", len(colls), colls)
	}

	// 5. Search
	searchBody, _ := json.Marshal(map[string]any{
		"query":      "password length symbols MFA",
		"collection": "security",
		"top_k":      3,
		"min_score":  -1.0,
	})
	reqSearch, _ := http.NewRequest(http.MethodPost, "/api/v1/knowledge/search", bytes.NewReader(searchBody))
	reqSearch.Header.Set("Content-Type", "application/json")
	reqSearch.Header.Set("Authorization", "Bearer "+token)
	wSearch := httptest.NewRecorder()
	router.ServeHTTP(wSearch, reqSearch)

	if wSearch.Code != http.StatusOK {
		t.Fatalf("search failed: %d: %s", wSearch.Code, wSearch.Body.String())
	}

	var searchResp struct {
		MatchesCount int                       `json:"matches_count"`
		Results      []*store.KnowledgeCitation `json:"results"`
	}
	if err := json.Unmarshal(wSearch.Body.Bytes(), &searchResp); err != nil {
		t.Fatalf("unmarshal search response failed: %v", err)
	}
	if searchResp.MatchesCount == 0 || len(searchResp.Results) == 0 {
		t.Fatalf("expected search results, got 0")
	}
	if searchResp.Results[0].Citation != "[IT_Security_Policy.txt, Page 1]" {
		t.Errorf("unexpected citation: %s", searchResp.Results[0].Citation)
	}

	// 6. Delete document
	reqDel, _ := http.NewRequest(http.MethodDelete, "/api/v1/knowledge/documents/"+createdDoc.ID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+token)
	wDel := httptest.NewRecorder()
	router.ServeHTTP(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("delete failed: %d", wDel.Code)
	}
}
