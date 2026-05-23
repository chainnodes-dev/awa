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

clear
echo -e "${BOLD}${CYAN}====================================================${NC}"
echo -e "${BOLD}${CYAN}            PHAXA ENTERPRISE LICENSE GENERATOR       ${NC}"
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
TENANT=""
TIER="enterprise"
DAYS="365"
KEY_PATH="$DEFAULT_KEY"
FEATURES="sso,branding,secrets,audit_logs,analytics"

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
        -t|--tenant) TENANT="$2"; shift ;;
        -r|--tier) TIER="$2"; shift ;;
        -d|--days) DAYS="$2"; shift ;;
        -k|--key) KEY_PATH="$2"; shift ;;
        -f|--features) FEATURES="$2"; shift ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  -g, --gen-key          Generate a new RSA 2048-bit Private/Public Key Pair"
            echo "  -t, --tenant <id>      Target Customer Tenant ID (Required for signing)"
            echo "  -r, --tier <tier>      License Tier (pro, enterprise) (Default: enterprise)"
            echo "  -d, --days <days>      Validity duration in days (Default: 365)"
            echo "  -k, --key <path>       Path to RSA private key file (Default: private.pem)"
            echo "  -f, --features <list>  Comma-separated list of enabled features (Default: sso,branding,secrets,audit_logs,analytics)"
            exit 0
            ;;
        *) error_exit "Unknown parameter passed: $1. Use --help for usage details." ;;
    esac
    shift
done

# If no tenant provided via args, run the interactive wizard
if [ -z "$TENANT" ]; then
    echo -e "${BOLD}${YELLOW}Entering interactive license setup wizard...${NC}"
    echo ""

    # Prompt Tenant ID
    while [ -z "$TENANT" ]; do
        read -p "$(echo -e ${BOLD}${CYAN}"🔑 Enter Customer Tenant ID (e.g. company_corp): "${NC})" TENANT
        if [ -z "$TENANT" ]; then
            echo -e "${RED}Tenant ID cannot be empty. Please enter a valid ID.${NC}"
        fi
    done

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
go run ./cmd/license-gen/main.go \
    -key "$KEY_PATH" \
    -tenant "$TENANT" \
    -tier "$TIER" \
    -days "$DAYS" \
    -features "$FEATURES"

echo ""
echo -e "${GREEN}${BOLD}✔ License successfully compiled and signed!${NC}"
echo -e "${BLUE}Copy the token above and paste it into Phaxa under Usage & Reporting -> Update Key.${NC}"
