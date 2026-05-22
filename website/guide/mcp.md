# MCP Integration

Chain Nodes is a native **Model Context Protocol (MCP)** host. It can connect to any MCP-compliant server to give its agents "hands" to interact with the world.

## How it Works
1. **Registry**: Servers are defined in `mcp_registry.yaml`.
2. **Discovery**: When an agent starts, Chain Nodes connects to the specified MCP servers and discovers available tools.
3. **Execution**: The LLM chooses a tool, and Chain Nodes executes the call securely.

## Configuring a Server
Add your server to `mcp_registry.yaml`:

```yaml
  - name: google-sheets
    description: "Manage Google Sheets."
    env_var: GOOGLE_SHEETS_URL # If using SSE
```

## Supported Transports
- **Stdio**: Local processes (node, python, npx, etc.). Supported natively by default.
- **SSE**: Remote servers over HTTP.

## The Marketplace
Use the **Marketplace** in the Chain Nodes dashboard to discover and install community servers with one click. You can also customize your own server settings directly from the UI registry screen.
