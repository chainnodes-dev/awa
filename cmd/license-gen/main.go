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
	company := flag.String("company", "", "Customer / Organization Name (e.g. Acme Corp)")
	tenantID := flag.String("tenant", "*", "Target Tenant ID (* for any single-tenant deployment, or specific UUID for SaaS)")
	tier := flag.String("tier", "enterprise", "License tier (community, pro, enterprise)")
	days := flag.Int("days", 365, "Validity in days")
	features := flag.String("features", "sso,branding,secrets,audit_logs,analytics", "Comma-separated list of enabled features")
	outputPath := flag.String("out", "", "Optional path to save the license file (e.g. license.pem or customer.lic)")
	flag.Parse()

	if *company == "" && (*tenantID == "" || *tenantID == "*") {
		fmt.Println("Error: Either -company (Customer Name) or -tenant (Tenant ID) must be provided")
		flag.Usage()
		os.Exit(1)
	}
	if *tenantID == "" {
		*tenantID = "*"
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
		Tier:        enterprise.Tier(*tier),
		TenantID:    *tenantID,
		CompanyName: *company,
		IssuedAt:    time.Now().Unix(),
		ExpiresAt:   time.Now().AddDate(0, 0, *days).Unix(),
		Features:    strings.Split(*features, ","),
	}

	// 3. Sign
	token, err := signer.Sign(claims)
	if err != nil {
		fmt.Printf("Error signing license: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n--- CHAINNODES ENTERPRISE LICENSE TOKEN ---")
	fmt.Println(token)
	fmt.Println("--------------------------------------")
	if *company != "" {
		fmt.Printf("Licensed To: %s\n", *company)
	}
	if *tenantID == "*" {
		fmt.Println("Deployment:  Single-Tenant (Any Instance)")
	} else {
		fmt.Printf("Tenant ID:   %s (Multi-Tenant Restricted)\n", *tenantID)
	}
	fmt.Printf("Tier:        %s\n", *tier)
	fmt.Printf("Expires:     %s\n", time.Unix(claims.ExpiresAt, 0).Format(time.RFC822))
	fmt.Println("--------------------------------------")

	if *outputPath != "" {
		pemContent := fmt.Sprintf("-----BEGIN LICENSE-----\n%s\n-----END LICENSE-----\n", token)
		if err := os.WriteFile(*outputPath, []byte(pemContent), 0644); err != nil {
			fmt.Printf("Error writing license file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✔ License file saved to: %s\n", *outputPath)
	}
}
