package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

func (h *Handlers) GetMCPMarket(c *gin.Context) {
	source := h.cfg.MCPMarketSource

	var data []byte
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// Fetch from remote URL
		resp, fetchErr := httpClient.Get(source)
		if fetchErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch marketplace data from remote source: " + fetchErr.Error()})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Marketplace remote source returned status " + resp.Status})
			return
		}

		data, err = io.ReadAll(resp.Body)
	} else {
		// Read from local file
		data, err = os.ReadFile(source)
	}

	if err != nil {
		if os.IsNotExist(err) {
			// Return empty list if file doesn't exist yet
			c.JSON(http.StatusOK, []interface{}{})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read marketplace data: " + err.Error()})
		return
	}

	// Parse JSON to ensure it's valid before sending (optional but good for validation)
	var marketData interface{}
	if err := json.Unmarshal(data, &marketData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid JSON in marketplace source: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, marketData)
}
