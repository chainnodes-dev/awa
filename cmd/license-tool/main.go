package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Tier string

const (
	TierCommunity  Tier = "community"
	TierEnterprise Tier = "enterprise"
)

type LicenseClaims struct {
	Tier         Tier     `json:"tier"`
	TenantID     string   `json:"tenant_id"`
	IssuedAt     int64    `json:"iat"`
	ExpiresAt    int64    `json:"exp"`
	Features     []string `json:"features,omitempty"`
	jwt.RegisteredClaims
}

func main() {
	genKey := flag.Bool("gen-key", false, "Generate a new RSA key pair")
	genToken := flag.String("gen-token", "", "Generate a token for the given tier (community|enterprise)")
	tenantID := flag.String("tenant", "00000000-0000-0000-0000-000000000001", "Tenant ID for the token")
	privKeyPath := flag.String("priv-key", "license_private.pem", "Path to private key for signing")
	flag.Parse()

	if *genKey {
		generateKeys()
		return
	}

	if *genToken != "" {
		generateToken(Tier(*genToken), *tenantID, *privKeyPath)
		return
	}

	flag.Usage()
}

func generateKeys() {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate key: %v", err)
	}

	// Private Key
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	_ = os.WriteFile("license_private.pem", privPEM, 0600)

	// Public Key
	pubBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: pubBytes})
	_ = os.WriteFile("license_public.pem", pubPEM, 0644)

	fmt.Println("Keys generated:")
	fmt.Println("- license_private.pem (KEEP SECRET - used to sign tokens)")
	fmt.Println("- license_public.pem  (Add content to LICENSE_PUBLIC_KEY in .env)")
	fmt.Printf("\nPublic Key for .env:\n%s\n", string(pubPEM))
}

func generateToken(tier Tier, tenant string, privPath string) {
	privBytes, err := os.ReadFile(privPath)
	if err != nil {
		log.Fatalf("failed to read private key: %v", err)
	}
	block, _ := pem.Decode(privBytes)
	if block == nil {
		log.Fatal("failed to decode PEM block")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("failed to parse private key: %v", err)
	}

	claims := LicenseClaims{
		Tier:         tier,
		TenantID:     tenant,
		IssuedAt:     time.Now().Unix(),
		ExpiresAt:    time.Now().Add(365 * 24 * time.Hour).Unix(),
		Features:     []string{"sso", "branding", "secrets", "audit_logs", "analytics"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privKey)
	if err != nil {
		log.Fatalf("failed to sign token: %v", err)
	}

	fmt.Printf("\nGenerated %s Token for Tenant %s:\n%s\n", tier, tenant, tokenString)
}
