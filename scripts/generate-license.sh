#!/bin/bash

# Phaxa Enterprise License Key Generator CLI
# Built with premium styling and interactive wizard.

set -e

# Navigate to the repository root so relative paths work from anywhere
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Styling colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

clear 2>/dev/null || true
echo -e "${BOLD}${CYAN}====================================================${NC}"
echo -e "${BOLD}${CYAN}         CHAIN NODES ENTERPRISE LICENSE GENERATOR   ${NC}"
echo -e "${BOLD}${CYAN}====================================================${NC}"
echo ""

# Helper for showing errors
error_exit() {
    echo -e "${RED}${BOLD}Error: $1${NC}"
    exit 1
}

# 1. Check for private.pem or alternative key
DEFAULT_KEY="private.pem"
if [ ! -f "$DEFAULT_KEY" ]; then
    echo -e "${YELLOW}Warning: Default private key '$DEFAULT_KEY' not found in current directory.${NC}"
fi

# 2. Argument Parsing & Interactive Wizard
COMPANY=""
TENANT="*"
TIER="enterprise"
DAYS="365"
KEY_PATH="$DEFAULT_KEY"
FEATURES="sso,branding,secrets,audit_logs,analytics"
OUTPUT_FILE=""

# Parse arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -g|--gen-key) 
            echo -e "${BLUE}Generating a brand new RSA Key Pair...${NC}"
            go run ./cmd/license-tool/main.go -gen-key
            if [ -f "license_private.pem" ]; then
                cp license_private.pem private.pem
                echo -e "${GREEN}Copied license_private.pem to private.pem (default key).${NC}"
            fi
            echo -e "${GREEN}${BOLD}✔ Successfully generated new key pair!${NC}"
            exit 0
            ;;
        -c|--company) COMPANY="$2"; shift ;;
        -t|--tenant) TENANT="$2"; shift ;;
        -r|--tier) TIER="$2"; shift ;;
        -d|--days) DAYS="$2"; shift ;;
        -k|--key) KEY_PATH="$2"; shift ;;
        -f|--features) FEATURES="$2"; shift ;;
        -o|--out) OUTPUT_FILE="$2"; shift ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  -c, --company <name>   Customer / Organization Name (e.g. Acme Corp)"
            echo "  -t, --tenant <id>      Specific SaaS Tenant UUID (Default: * for single-tenant / any instance)"
            echo "  -r, --tier <tier>      License Tier (pro, enterprise) (Default: enterprise)"
            echo "  -d, --days <days>      Validity duration in days (Default: 365)"
            echo "  -k, --key <path>       Path to RSA private key file (Default: private.pem)"
            echo "  -f, --features <list>  Comma-separated list of enabled features (Default: sso,branding,secrets,audit_logs,analytics)"
            echo "  -o, --out <path>       Output path for the license file (e.g. license.pem or client.lic)"
            exit 0
            ;;
        *) error_exit "Unknown parameter passed: $1. Use --help for usage details." ;;
    esac
    shift
done

# If neither company nor specific tenant provided via args, run the interactive wizard
if [ -z "$COMPANY" ] && [ "$TENANT" == "*" ]; then
    echo -e "${BOLD}${YELLOW}Entering interactive license setup wizard...${NC}"
    echo ""

    # Prompt Company Name
    while [ -z "$COMPANY" ]; do
        read -p "$(echo -e ${BOLD}${CYAN}"🏢 Enter Customer / Organization Name (e.g. Acme Corp): "${NC})" COMPANY
        if [ -z "$COMPANY" ]; then
            echo -e "${RED}Company name cannot be empty. Please enter a valid name.${NC}"
        fi
    done

    # Optional Multi-Tenant UUID restriction
    read -p "$(echo -e ${BOLD}${CYAN}"🌐 Restrict to specific multi-tenant UUID? (y/N) [default: N]: "${NC})" is_saas
    if [[ "$is_saas" =~ ^[Yy]$ ]]; then
        read -p "$(echo -e ${BOLD}${CYAN}"🔑 Enter SaaS Tenant UUID: "${NC})" custom_tenant
        if [ ! -z "$custom_tenant" ]; then
            TENANT="$custom_tenant"
        fi
    fi

    # Prompt License Tier
    read -p "$(echo -e ${BOLD}${CYAN}"💎 Enter License Tier (pro | enterprise) [default: $TIER]: "${NC})" input_tier
    if [ ! -z "$input_tier" ]; then
        TIER="$input_tier"
    fi

    # Prompt Validity Duration
    read -p "$(echo -e ${BOLD}${CYAN}"⏳ Enter Validity Duration in days [default: $DAYS]: "${NC})" input_days
    if [ ! -z "$input_days" ]; then
        DAYS="$input_days"
    fi

    # Prompt Private Key Path
    read -p "$(echo -e ${BOLD}${CYAN}"🛡️ Enter Private Key Path [default: $KEY_PATH]: "${NC})" input_key
    if [ ! -z "$input_key" ]; then
        KEY_PATH="$input_key"
    fi

    # Prompt Features
    read -p "$(echo -e ${BOLD}${CYAN}"🛠️ Enter Features List [default: $FEATURES]: "${NC})" input_features
    if [ ! -z "$input_features" ]; then
        FEATURES="$input_features"
    fi
fi

# Validate private key exists
if [ ! -f "$KEY_PATH" ]; then
    error_exit "Private key file '$KEY_PATH' not found. Please generate or provide a valid RSA private key."
fi

echo ""
echo -e "${BLUE}Building license generation tool...${NC}"
# Compile / run the go tool
OUT_ARG=()
if [ ! -z "$OUTPUT_FILE" ]; then
    OUT_ARG=(-out "$OUTPUT_FILE")
fi

go run ./cmd/license-gen/main.go \
    -key "$KEY_PATH" \
    -company "$COMPANY" \
    -tenant "$TENANT" \
    -tier "$TIER" \
    -days "$DAYS" \
    -features "$FEATURES" \
    "${OUT_ARG[@]}"

echo ""
echo -e "${GREEN}${BOLD}✔ License successfully compiled and signed!${NC}"
echo -e "${BLUE}Deliver the license file/token to the customer to activate under Settings -> License / Usage.${NC}"
