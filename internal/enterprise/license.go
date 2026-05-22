package enterprise

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Tier string

const (
	TierFree       Tier = "free"
	TierCommunity  Tier = "community"
	TierPro        Tier = "pro"
	TierEnterprise Tier = "enterprise"

	// MonthlyLookback is the standard window for run quota enforcement.
	MonthlyLookback = 30 * 24 * time.Hour

	// ChainNodesPublicKey is the hardcoded public key for Phaxa license verification.
	// This ensures that only licenses signed by Chain Nodes SRL are accepted.
	ChainNodesPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA3QwlqfwEnHKdXXYG1aL/
dbAO7DsV7InutB6iub+onloy53BWvxmkJY80ZzOsn8IqaxEcqXoo69Iu4r5ooS+A
0Q/pZ8pab1COhnVq6Bv8ZA63l3ZI12qA10TA/FraHh/VCBkKUyJ1n/PbtJW4TtRr
PtG1/1pjHAXhGZmr6PrKwKHANNaOgPvZxr6HiizN/IWW1nGKO2OCbJMJAdKx25JA
VkqF2xXw4bB1uCHBfZ3tsJUqx2KDIt41IqDBgxrhjRery3aNzTlJGCJiTT/h5Jev
OEkVnz0tVTlgtghkHngwMRjZduvejFo4qZFNCjHgrGRdBtF1Dlfy22+ZI24/Weje
dQIDAQAB
-----END PUBLIC KEY-----`
)

// LicenseClaims represents the signed information in an enterprise license.
type LicenseClaims struct {
	Tier         Tier     `json:"tier"`
	TenantID     string   `json:"tenant_id"`
	IssuedAt     int64    `json:"iat"`
	ExpiresAt    int64    `json:"exp"`
	Features     []string `json:"features,omitempty"`
	jwt.RegisteredClaims
}

var (
	ErrLicenseExpired    = errors.New("license has expired")
	ErrInvalidLicense    = errors.New("invalid license token")
	ErrTierNotMet        = errors.New("required tier not met")
	ErrFeatureNotEnabled = errors.New("feature not enabled for this license")
)

// Verifier handles license validation using a trusted RSA public key.
type Verifier struct {
	publicKey *rsa.PublicKey
}

func NewVerifier(pubKey *rsa.PublicKey) *Verifier {
	return &Verifier{publicKey: pubKey}
}

// Signer handles license generation using an RSA private key.
type Signer struct {
	privateKey *rsa.PrivateKey
}

func NewSigner(privKey *rsa.PrivateKey) *Signer {
	return &Signer{privateKey: privKey}
}

func (s *Signer) Sign(claims *LicenseClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

// Verify decodes and validates a signed JWT license token.
func (v *Verifier) Verify(tokenString string) (*LicenseClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &LicenseClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		// Use the configured public key if available, otherwise fallback to the hardcoded Chain Nodes key.
		if v != nil && v.publicKey != nil {
			return v.publicKey, nil
		}

		return ParseRSAPublicKey(ChainNodesPublicKey)
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidLicense, err)
	}

	if claims, ok := token.Claims.(*LicenseClaims); ok && token.Valid {
		if time.Now().Unix() > claims.ExpiresAt {
			return nil, ErrLicenseExpired
		}
		return claims, nil
	}

	return nil, ErrInvalidLicense
}

// HasFeature checks if the license allows a specific named capability.
func (c *LicenseClaims) HasFeature(feature string) bool {
	if c.Tier == TierEnterprise {
		return true // Enterprise has everything
	}
	for _, f := range c.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// DefaultFreeLicense returns a standard set of claims for unmanaged tenants.
func DefaultFreeLicense(tenantID string) *LicenseClaims {
	return &LicenseClaims{
		Tier:         TierFree,
		TenantID:     tenantID,
		Features:     []string{"basic_execution"},
	}
}

// ParseRSAPublicKey parses a PEM-encoded RSA public key.
func ParseRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	return rsaPub, nil
}

// ParseRSAPrivateKey parses a PEM-encoded RSA private key.
func ParseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Fallback to PKCS8
		p8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		rsaPriv, ok := p8.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}
		return rsaPriv, nil
	}
	return priv, nil
}
// GenerateRSAKeyPair creates a new 2048-bit RSA key pair and returns them as PEM strings.
func GenerateRSAKeyPair() (privPEM string, pubPEM string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// Private Key
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	privPEM = string(pem.EncodeToMemory(privBlock))

	// Public Key
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", err
	}
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pubPEM = string(pem.EncodeToMemory(pubBlock))

	return privPEM, pubPEM, nil
}
