# MCPRelay
MCPRelay allows MCP clients that only support stdio to connect to SSE servers. Note that non-SSE HTTP servers are not currently supported.

Example configuration for Claude desktop:


```
{
    "mcpServers": {
        "MCPLaunchPad": {
            "command": "/opt/MCPRelay/mcprelay",
            "args": [
                "-url",
                "http://127.0.0.1:8888/sse",
                "-log",
                "/tmp/relay1.log",
                "-debug"
            ]
        },
        "AnotherServer": {
            "command": "/opt/MCPRelay/mcprelay",
            "args": [
                "-url",
                "http://127.0.0.1:8889/sse",
                "-log",
                "/tmp/relay2.log",
                "-debug"
            ]
        }
    }
}
```

NOTES:
- Specify the URL to the SSE endpoint. The server will tell MCPRelay, acting as a client, what URL to POST requests to.
- Multiple instances are perfectly fine. Your MCP server will start a separate instance of each and communicate with it over stdin/stdout. You may wish to specify a different log file for each instance.