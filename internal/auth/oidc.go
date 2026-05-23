package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// IDPConfig represents an external Identity Provider configuration for a tenant.
type IDPConfig struct {
	TenantID     string            `json:"tenant_id"`
	IssuerURL    string            `json:"issuer_url"`
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"client_secret"`
	RoleMapping  map[string]string `json:"role_mapping"` // Map of IDP group names to Phaxa roles
	Active       bool              `json:"active"`
}

// OIDCProvider encapsulates the logic for OIDC authentication.
type OIDCProvider struct {
	config *IDPConfig
}

func NewOIDCProvider(cfg *IDPConfig) *OIDCProvider {
	return &OIDCProvider{config: cfg}
}

// OpenIDConfiguration represents the metadata fetched from the discovery endpoint.
type OpenIDConfiguration struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

func (p *OIDCProvider) Discover(ctx context.Context) (*OpenIDConfiguration, error) {
	url := fmt.Sprintf("%s/.well-known/openid-configuration", p.config.IssuerURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery failed with status %d", resp.StatusCode)
	}

	var config OpenIDConfiguration
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ExchangeToken swaps an authorization code for an ID token.
func (p *OIDCProvider) ExchangeToken(ctx context.Context, code string, redirectURI string) (map[string]interface{}, error) {
	disc, err := p.Discover(ctx)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Add("grant_type", "authorization_code")
	form.Add("code", code)
	form.Add("redirect_uri", redirectURI)
	form.Add("client_id", p.config.ClientID)
	form.Add("client_secret", p.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", disc.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// VerifyIDToken parses an ID token and returns its claims.
// This is a minimal implementation for demonstration; production use should
// verify signatures against the JWKS provider.
func (p *OIDCProvider) VerifyIDToken(tokenStr string) (jwt.MapClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid claims type")
}
