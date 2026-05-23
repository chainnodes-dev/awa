package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/store"
)

// generateAPIKey produces a new raw API key in the format phaxa_sk_<base64url(32 bytes)>.
// The key is shown once at creation time; only its SHA-256 hash is stored.
func generateAPIKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return "phaxa_sk_" + base64.RawURLEncoding.EncodeToString(b)
}

func (h *Handlers) CreateAPIKey(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	tenantID := store.TenantIDFromContext(ctx)

	var createdBy string
	if claims := auth.ClaimsFrom(c); claims != nil {
		createdBy = claims.UserID
	}

	rawKey := generateAPIKey()
	key := &store.APIKey{
		TenantID:  tenantID,
		Name:      body.Name,
		KeyHash:   hashAPIKey(rawKey),
		CreatedBy: createdBy,
	}
	if err := h.store.CreateAPIKey(ctx, key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// raw_key is returned only once — never stored in plaintext.
	c.JSON(http.StatusCreated, gin.H{"key": key, "raw_key": rawKey})
}

func (h *Handlers) ListAPIKeys(c *gin.Context) {
	keys, err := h.store.ListAPIKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, keys)
}

func (h *Handlers) RevokeAPIKey(c *gin.Context) {
	if err := h.store.RevokeAPIKey(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
