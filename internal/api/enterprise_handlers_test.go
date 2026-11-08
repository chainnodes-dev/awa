package api
 
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/enterprise"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/health"
	"github.com/asm-platform/asm/internal/mcp"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/store"
)

func TestSetLicense_UploadFormats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Generate RSA key pair for testing
	privPEM, pubPEM, err := enterprise.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	privKey, err := enterprise.ParseRSAPrivateKey(privPEM)
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}

	pubKey, err := enterprise.ParseRSAPublicKey(pubPEM)
	if err != nil {
		t.Fatalf("failed to parse public key: %v", err)
	}

	verifier := enterprise.NewVerifier(pubKey)
	signer := enterprise.NewSigner(privKey)

	// 2. Setup mock dependencies
	s := store.NewMemoryStore()
	bus := events.NewLocalBus()
	temporal := &stubTemporalClient{}
	eng := orchestrator.NewEngine(s, bus, temporal)
	jwtSvc := auth.NewJWTService("test-secret")
	mcpMgr := mcp.NewManager(s)

	// Create tenant and admin user
	tenantID := store.DefaultTenantID
	tenant := &store.Tenant{
		ID:   tenantID,
		Name: "Test Tenant",
		Slug: "default",
	}
	if err := s.CreateTenant(context.Background(), tenant); err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	adminToken, err := jwtSvc.GenerateAccessToken("test-admin-id", tenantID, "admin", string(auth.RoleAdmin))
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	authHeader := "Bearer " + adminToken

	// Instantiate handlers with our verifier and signer
	handlers := NewHandlers(&config.Config{}, eng, s, jwtSvc, nil, nil, nil, nil, verifier, signer, mcpMgr, nil, nil)
	router := gin.New()
	registerRoutes(router, handlers, NewHub(bus), jwtSvc, health.New())

	// Generate a valid license token
	claims := &enterprise.LicenseClaims{
		Tier:      enterprise.TierEnterprise,
		TenantID:  tenantID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}
	token, err := signer.Sign(claims)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// --- TEST CASE 1: Standard JSON Upload ---
	t.Run("JSON Request", func(t *testing.T) {
		body := jsonBody(t, map[string]string{"token": token})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/enterprise/license", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
		}

		// Verify saved in DB
		gotTenant, err := s.GetTenant(context.Background(), tenantID)
		if err != nil || gotTenant.LicenseToken != token {
			t.Errorf("license token not saved correctly: %v", err)
		}
	})

	// Clear license from DB for next test
	_ = s.UpdateTenantLicense(context.Background(), tenantID, "")

	// --- TEST CASE 2: Raw Text File Upload ---
	t.Run("Raw Text File", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "license.lic")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		part.Write([]byte(token))
		writer.Close()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/enterprise/license", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
		}

		gotTenant, _ := s.GetTenant(context.Background(), tenantID)
		if gotTenant.LicenseToken != token {
			t.Errorf("license token not matching: got %q, want %q", gotTenant.LicenseToken, token)
		}
	})

	// Clear license from DB for next test
	_ = s.UpdateTenantLicense(context.Background(), tenantID, "")

	// --- TEST CASE 3: PEM-wrapped File Upload ---
	t.Run("PEM Wrapped File", func(t *testing.T) {
		pemContent := fmt.Sprintf("-----BEGIN LICENSE-----\n%s\n-----END LICENSE-----", token)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("license", "license.pem") // test fallback field name "license"
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		part.Write([]byte(pemContent))
		writer.Close()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/enterprise/license", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
		}

		gotTenant, _ := s.GetTenant(context.Background(), tenantID)
		if gotTenant.LicenseToken != token {
			t.Errorf("failed to extract and save token: got %q, want %q", gotTenant.LicenseToken, token)
		}
	})

	// --- TEST CASE 4: Wildcard Single-Tenant License with Company Name ---
	t.Run("Wildcard Single-Tenant License with Company Name", func(t *testing.T) {
		wildcardClaims := &enterprise.LicenseClaims{
			Tier:        enterprise.TierEnterprise,
			TenantID:    "*",
			CompanyName: "Acme Global Logistics",
			IssuedAt:    time.Now().Unix(),
			ExpiresAt:   time.Now().Add(24 * time.Hour).Unix(),
			Features:    []string{"sso", "branding"},
		}
		wildcardToken, err := signer.Sign(wildcardClaims)
		if err != nil {
			t.Fatalf("failed to sign wildcard token: %v", err)
		}

		body := jsonBody(t, map[string]string{"token": wildcardToken})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/enterprise/license", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for wildcard license, got %d (body: %s)", w.Code, w.Body.String())
		}

		// Verify GET /api/v1/enterprise/status returns company name
		statusW := httptest.NewRecorder()
		statusReq, _ := http.NewRequest("GET", "/api/v1/enterprise/status", nil)
		statusReq.Header.Set("Authorization", authHeader)
		router.ServeHTTP(statusW, statusReq)

		if statusW.Code != http.StatusOK {
			t.Fatalf("expected 200 from status, got %d", statusW.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(statusW.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse status json: %v", err)
		}

		if resp["company_name"] != "Acme Global Logistics" {
			t.Errorf("expected company_name 'Acme Global Logistics', got %v", resp["company_name"])
		}
		if resp["tier"] != "enterprise" {
			t.Errorf("expected tier 'enterprise', got %v", resp["tier"])
		}
	})

	// --- TEST CASE 5: Tenant Mismatch Rejected ---
	t.Run("Tenant ID Mismatch Rejected", func(t *testing.T) {
		mismatchedClaims := &enterprise.LicenseClaims{
			Tier:      enterprise.TierEnterprise,
			TenantID:  "different-tenant-uuid",
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		}
		mismatchedToken, err := signer.Sign(mismatchedClaims)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		body := jsonBody(t, map[string]string{"token": mismatchedToken})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/enterprise/license", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for mismatched tenant, got %d", w.Code)
		}
	})
}
