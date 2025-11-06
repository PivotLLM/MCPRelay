# MCPRelay
MCPRelay allows MCP clients that only support stdio to connect to SSE servers. Note that non-SSE HTTP servers are not currently supported.

## Command-line Options
- `-url`: URL to connect to SSE stream (default: `http://127.0.0.1:8888/sse`)
- `-log`: Path to the log file (leave empty to disable logging)
- `-debug`: Enable debug logging
- `-headers`: Custom HTTP headers as JSON object (e.g., `'{"Authorization":"Bearer token"}'`)

### Example configuration for Claude desktop:
```
{
  "mcpServers": {
    "Fusion": {
      "command": "/opt/mcprelay/mcprelay",
      "args": [
        "-url",
        "http://127.0.0.1:8888/sse",
        "-headers",
        "{\"Authorization\":\"Bearer <token>\"}",
        "-log",
        "/opt/mcprelay/relay-fusion.log",
        "-debug"
      ]
    }
  }
}
```

### Example configuration for Cline:
```
{
  "mcpServers": {
    "ServerName": {
      "disabled": false,
      "timeout": 3600,
      "command": "/opt/MCPRelay/mcprelay",
      "args": [
        "-url",
        "http://127.0.0.1:8888/sse",
        "-log",
        "/tmp/relay.log",
        "-debug"
      ]
      "transportType": "stdio"
    }
  }
}
```

### Example with custom headers:
```
{
  "mcpServers": {
    "ServerWithAuth": {
      "command": "/opt/MCPRelay/mcprelay",
      "args": [
        "-url",
        "http://127.0.0.1:8888/sse",
        "-headers",
        "{\"Authorization\":\"Bearer your-token-here\",\"X-Custom-Header\":\"value\"}",
        "-log",
        "/tmp/relay.log"
      ]
    }
  }
}
```

### NOTES:
- Specify the URL to the SSE endpoint. The server will tell MCPRelay, acting as a client, what URL to POST requests to.
- Multiple instances are perfectly fine. Your MCP server will start a separate instance of each and communicate with it over stdin/stdout. You may wish to specify a different log file for each instance.
- All arguments are optional. If you don't specify an endpoint, it will default to `http://127.0.0.1:8888/sse`.
- Custom headers specified with `-headers` will be sent with every HTTP request (both SSE connections and POST requests).
