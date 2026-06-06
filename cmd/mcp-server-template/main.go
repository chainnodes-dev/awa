// cmd/mcp-server-template is a ready-to-copy starter for building standalone MCP servers.
//
// # Dockerfile (copy this):
//
//	FROM golang:1.23-alpine AS builder
//	WORKDIR /app
//	COPY . .
//	RUN go build -o mcp-server .
//
//	FROM alpine:latest
//	COPY --from=builder /app/mcp-server .
//	EXPOSE 8090
//	CMD ["./mcp-server"]
//
// # Usage
//
//  1. Copy this directory to your own repository.
//  2. Update the module path in go.mod.
//  3. Replace the example tool with your own logic.
//  4. Run: PORT=8090 go run .
//
// # Environment variables
//
//	PORT  HTTP port to listen on (default: 8090)
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/asm-platform/asm/internal/mcp/server"
)

func main() {
	s := server.New("mcp-server-template", "1.0.0")

	s.Register(server.Tool{
		Name:        "greet",
		Description: "Returns a greeting for the given name.",
		InputSchema: server.InputSchema{
			Type: "object",
			Properties: map[string]server.ToolParam{
				"name": {Type: "string", Description: "The name to greet"},
			},
			Required: []string{"name"},
		},
		Handler: greetHandler,
	})

	if err := s.Start("8090"); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func greetHandler(args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return fmt.Sprintf("Hello, %s! 👋", name), nil
}
