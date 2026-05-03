#!/bin/bash

# ASM Trading MCP Setup & Test Script
# This script helps you verify your API keys and provides the commands for the ASM Dashboard.

echo "------------------------------------------------"
echo "📈 ASM Crypto Sentinel: MCP Setup Utility"
echo "------------------------------------------------"

# 1. Check for API Keys
if [ -z "$BRAVE_SEARCH_API_KEY" ]; then
    echo "⚠️  WARNING: BRAVE_SEARCH_API_KEY is not set."
    echo "   Get one at: https://api.search.brave.com/"
else
    echo "✅ BRAVE_SEARCH_API_KEY found."
fi

if [ -z "$COINGECKO_API_KEY" ]; then
    echo "⚠️  WARNING: COINGECKO_API_KEY is not set (Demo keys are free)."
    echo "   Get one at: https://www.coingecko.com/en/developers/dashboard"
else
    echo "✅ COINGECKO_API_KEY found."
fi

echo ""
echo "------------------------------------------------"
echo "📋 Registration Commands for ASM Dashboard"
echo "------------------------------------------------"
echo "Copy and paste these into Settings > MCP Servers:"
echo ""
echo "CoinGecko Server:"
echo "npx -y @modelcontextprotocol/server-coingecko"
echo ""
echo "Brave Search Server:"
echo "npx -y @modelcontextprotocol/server-brave-search"
echo ""
echo "------------------------------------------------"
echo "🚀 Verification: Starting servers for 5s..."
echo "------------------------------------------------"

# Test CoinGecko
echo "Testing CoinGecko connection..."
timeout 5 npx -y @modelcontextprotocol/server-coingecko > /dev/null 2>&1
if [ $? -eq 124 ]; then
    echo "✅ CoinGecko server started successfully (Timed out as expected)."
else
    echo "❌ CoinGecko failed to start. Check your Node.js/npx installation."
fi

# Test Brave Search
if [ ! -z "$BRAVE_SEARCH_API_KEY" ]; then
    echo "Testing Brave Search connection..."
    timeout 5 npx -y @modelcontextprotocol/server-brave-search > /dev/null 2>&1
    if [ $? -eq 124 ]; then
        echo "✅ Brave Search server started successfully."
    else
        echo "❌ Brave Search failed to start. Check your API key."
    fi
fi

echo ""
echo "Done. If everything is green, register them in the ASM UI and start your Sentinel!"
