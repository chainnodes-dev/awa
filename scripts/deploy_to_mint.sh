#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

MINT_USER="${1:-egon}"
MINT_HOST="${2:-mintbox}"
REMOTE_TARGET="/home/${MINT_USER}/apps/asm-platform"
SYNC_ONLY="${3:-}"

[[ "$MINT_USER" =~ ^[a-z_][a-z0-9_-]*$ && "$MINT_HOST" =~ ^[a-zA-Z0-9][a-zA-Z0-9.-]*$ ]] || { echo "Invalid SSH destination"; exit 1; }

echo "========================================================="
echo " 🚀 Chain Nodes Platform: Linux Mint Deployment"
echo " Target: ${MINT_USER}@${MINT_HOST}:${REMOTE_TARGET}"
echo "========================================================="

# 1. Create remote target directory
echo "📁 Ensuring remote directories exist..."
ssh "${MINT_USER}@${MINT_HOST}" "mkdir -p ${REMOTE_TARGET}/data ${REMOTE_TARGET}/temporal-config"

# 2. Sync project source code to mintbox
echo "📦 Syncing source files to ${MINT_HOST}..."
rsync -avz --delete \
  --include '.env.mint.example' \
  --include '.env.example' \
  --exclude '.env*' \
  --exclude '.git' \
  --exclude '.cache' \
  --exclude 'venv' \
  --exclude '.venv' \
  --exclude 'website/venv' \
  --exclude 'node_modules' \
  --exclude 'dist' \
  --exclude 'tmp' \
  --exclude 'data/*' \
  --exclude '*.log' \
  --exclude '*.pem' \
  --exclude 'private*' \
  --exclude 'cmd/license-gen' \
  --exclude 'scripts/generate-license.sh' \
  --exclude '.DS_Store' \
  --exclude '.idea' \
  --exclude '.vscode' \
  --exclude '.contextshrinker' \
  ./ "${MINT_USER}@${MINT_HOST}:${REMOTE_TARGET}/"

# 3. Initialize .env on mintbox if not already present
echo "⚙️  Checking environment configuration..."
ssh "${MINT_USER}@${MINT_HOST}" "cd ${REMOTE_TARGET} && if [ ! -f .env ]; then cp .env.mint.example .env; echo 'Initialized .env from .env.mint.example'; fi"

if [[ "$SYNC_ONLY" == "--sync-only" ]]; then
  echo "✅ Source synchronized; containers were not restarted."
  exit 0
fi

# 4. Build & launch Docker stack on mintbox
echo "🐳 Building and starting Docker containers on ${MINT_HOST}..."
ssh "${MINT_USER}@${MINT_HOST}" "cd ${REMOTE_TARGET} && docker compose -f docker-compose.mint.yml up -d --build"

# 5. Output endpoints
echo "========================================================="
echo " 🎉 DEPLOYMENT SUCCESSFUL ON MINTBOX!"
echo "========================================================="
echo ""
echo "Local LAN Endpoints (Accessible within your home network):"
echo "  • Frontend:     http://${MINT_HOST}:5174"
echo "  • Backend API:  http://${MINT_HOST}:8080"
echo "  • Temporal UI:  http://${MINT_HOST}:8088"
echo "  • PostgreSQL:   ${MINT_HOST}:5435"
echo ""
echo "Remote Cloudflare Access:"
echo "  If CLOUDFLARE_TUNNEL_TOKEN is configured in ${REMOTE_TARGET}/.env,"
echo "  the tunnel will be active and forwarding to the frontend."
echo "========================================================="
