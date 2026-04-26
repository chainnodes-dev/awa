package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/enterprise"
)

// GetSystemKeys returns the platform's licensing keys.
func (h *Handlers) GetSystemKeys(c *gin.Context) {
	pubKey, _ := h.store.GetSystemSetting(c.Request.Context(), "license_public_key")
	privKey, _ := h.store.GetSystemSetting(c.Request.Context(), "license_private_key")

	// Mask the private key for security.
	maskedPriv := ""
	if privKey != "" {
		lines := strings.Split(privKey, "\n")
		if len(lines) > 2 {
			maskedPriv = lines[0] + "\n" + "**** MASKED ****" + "\n" + lines[len(lines)-1]
		} else {
			maskedPriv = "**** MASKED ****"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"public_key":  pubKey,
		"private_key": maskedPriv,
	})
}

// UpdateSystemKeys sets the platform's licensing keys.
func (h *Handlers) UpdateSystemKeys(c *gin.Context) {
	var req struct {
		PublicKey  string `json:"public_key" binding:"required"`
		PrivateKey string `json:"private_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate keys before saving.
	if _, err := enterprise.ParseRSAPublicKey(req.PublicKey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid public key: %v", err)})
		return
	}
	if _, err := enterprise.ParseRSAPrivateKey(req.PrivateKey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid private key: %v", err)})
		return
	}

	if err := h.store.SetSystemSetting(c.Request.Context(), "license_public_key", req.PublicKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSystemSetting(c.Request.Context(), "license_private_key", req.PrivateKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Hot reload.
	_ = h.ReloadLicenseKeys(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "platform keys updated"})
}

// GenerateSystemKeys generates a new RSA key pair for the platform.
func (h *Handlers) GenerateSystemKeys(c *gin.Context) {
	priv, pub, err := enterprise.GenerateRSAKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate keys: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"public_key":  pub,
		"private_key": priv,
	})
}
