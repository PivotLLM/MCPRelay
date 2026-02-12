# MCPRelay
MCPRelay allows MCP clients that only support stdio to connect to network MCP servers using either HTTP or SSE transport.

It was originally developed to address desktop AI clients with missing or limited network MCP capabilities.

## Command-line Options
- `-url`: URL to connect to (default: `http://127.0.0.1:8888/sse`)
  - For HTTP mode: POST endpoint (e.g., `http://127.0.0.1:9999/mcp`)
  - For SSE mode: SSE stream endpoint (e.g., `http://127.0.0.1:8888/sse`)
- `-transport`: Transport mode - `http` or `sse` (default: `http`)
- `-log`: Path to the log file (leave empty to disable logging)
- `-debug`: Enable debug logging
- `-headers`: Custom HTTP headers as JSON object (e.g., `'{"Authorization":"Bearer token"}'`)

### Example configuration for HTTP transport (Claude desktop):
```
{
  "mcpServers": {
    "Fusion": {
      "command": "/opt/mcprelay/mcprelay",
      "args": [
        "-url",
        "http://127.0.0.1:9999/mcp",
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

### Example configuration for SSE transport (Claude desktop):
```
{
  "mcpServers": {
    "Fusion": {
      "command": "/opt/mcprelay/mcprelay",
      "args": [
        "-transport",
        "sse",
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

### Example configuration for Cline (HTTP mode):
```
{
  "mcpServers": {
    "ServerName": {
      "disabled": false,
      "timeout": 3600,
      "command": "/opt/MCPRelay/mcprelay",
      "args": [
        "-url",
        "http://127.0.0.1:9999/mcp",
        "-log",
        "/tmp/relay.log",
        "-debug"
      ],
      "transportType": "stdio"
    }
  }
}
```

### NOTES:
- **HTTP mode (default)**: Specify the POST endpoint URL. Each message is sent via POST and receives an immediate response. This is the modern, stateless transport.
- **SSE mode**: Specify the SSE stream URL with `-transport sse`. The server will tell MCPRelay what URL to POST requests to via dynamic endpoint discovery.
- Multiple instances are perfectly fine. Your MCP client will start a separate instance and communicate with it over stdin/stdout. You may wish to specify a different log file for each instance.
- All arguments are optional. Default transport is `http` and default URL is `http://127.0.0.1:8888/sse`.
- Custom headers specified with `-headers` will be sent with every HTTP request (both SSE connections and POST requests).

## Copyright and License

Copyright (c) 2025-2026 by Tenebris Technologies Inc. This software is licensed under the MIT License. Please see LICENSE for details.

## Trademarks

Any trademarks referenced are the propery of their respective owners, used for identification only, and do not imply sponsorship, endorsement, or affiliation.

## No Warranty (zilch, none, void, nil, null, "", {}, 0x00, 0b00000000, EOF)

THIS SOFTWARE IS PROVIDED “AS IS,” WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND NON-INFRINGEMENT. IN NO EVENT SHALL THE COPYRIGHT HOLDERS OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

Made in Canada
