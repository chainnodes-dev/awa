# MCP Integration

Phaxa is a native **Model Context Protocol (MCP)** host. It can connect to any MCP-compliant server to give its agents "hands" to interact with the world.

## How it Works
1. **Registry**: Servers are defined in `mcp_registry.yaml`.
2. **Discovery**: When an agent starts, Phaxa connects to the specified MCP servers and discovers available tools.
3. **Execution**: The LLM chooses a tool, and Phaxa executes the call securely.

## Configuring a Server
Add your server to `mcp_registry.yaml`:

```yaml
  - name: google-sheets
    description: "Manage Google Sheets."
    env_var: GOOGLE_SHEETS_URL # If using SSE
```

## Supported Transports
- **Stdio**: Local processes (npx, python, etc.). Enabled via `PHAXA_ALLOW_STDIO_MCP=true`.
- **SSE**: Remote servers over HTTP.

## The Marketplace
Use the **Marketplace** in the Phaxa dashboard to discover and "Install" community servers with one click.
