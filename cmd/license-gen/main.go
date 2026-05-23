package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/asm-platform/asm/internal/enterprise"
)

func main() {
	keyPath := flag.String("key", "private.pem", "Path to the RSA private key file")
	tenantID := flag.String("tenant", "", "The target Customer Tenant ID")
	tier := flag.String("tier", "enterprise", "License tier (community, pro, enterprise)")
	days := flag.Int("days", 365, "Validity in days")
	features := flag.String("features", "sso,branding,secrets,audit_logs,analytics", "Comma-separated list of enabled features")
	flag.Parse()

	if *tenantID == "" {
		fmt.Println("Error: -tenant ID is required")
		flag.Usage()
		os.Exit(1)
	}

	// 1. Load Private Key
	keyBytes, err := os.ReadFile(*keyPath)
	if err != nil {
		fmt.Printf("Error reading key file: %v\n", err)
		os.Exit(1)
	}

	privKey, err := enterprise.ParseRSAPrivateKey(string(keyBytes))
	if err != nil {
		fmt.Printf("Error parsing private key: %v\n", err)
		os.Exit(1)
	}

	// 2. Prepare Claims
	signer := enterprise.NewSigner(privKey)
	claims := &enterprise.LicenseClaims{
		Tier:      enterprise.Tier(*tier),
		TenantID:  *tenantID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().AddDate(0, 0, *days).Unix(),
		Features:  strings.Split(*features, ","),
	}

	// 3. Sign
	token, err := signer.Sign(claims)
	if err != nil {
		fmt.Printf("Error signing license: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n--- PHAXA ENTERPRISE LICENSE TOKEN ---")
	fmt.Println(token)
	fmt.Println("--------------------------------------")
	fmt.Printf("Issued To: %s\n", *tenantID)
	fmt.Printf("Tier:      %s\n", *tier)
	fmt.Printf("Expires:   %s\n", time.Unix(claims.ExpiresAt, 0).Format(time.RFC822))
	fmt.Println("--------------------------------------")
}
