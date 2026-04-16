#!/usr/bin/env bash

# ASM Platform — Start Demo MCP Servers
# This script starts the 3 dedicated demo MCP servers in the background
# and ties them to this terminal process so they can be stopped cleanly.

# 1. Start the GLEIF Server (Port 8091)
echo "Starting GLEIF MCP Server on port 8091..."
go run ./cmd/mcp-gleif &
PID1=$!

# 2. Start the Exchange Rates Server (Port 8092)
echo "Starting Exchange Rates MCP Server on port 8092..."
go run ./cmd/mcp-exchange &
PID2=$!

# 3. Start the OpenFIGI Server (Port 8093)
echo "Starting OpenFIGI MCP Server on port 8093..."
go run ./cmd/mcp-openfigi &
PID3=$!

echo "=========================================================="
echo "✅ All 3 Demo MCP Servers are running in the background."
echo "🛑 Press Ctrl+C to stop all three servers cleanly."
echo "=========================================================="

# Trap SIGINT (Ctrl+C) and SIGTERM to kill the background processes
trap "echo -e '\nReceived shutdown signal. Stopping MCP servers...'; kill $PID1 $PID2 $PID3; exit" SIGINT SIGTERM

# Wait indefinitely until interrupted
wait
