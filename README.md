# MCPRelay
MCPRelay allows MCP clients that only support stdio to connect to network MCP servers using SSE transport. Note that non-SSE HTTP servers are not yet supported.

It was originally developed to address desktop AI clients with missing or limited network MCP capabilities.

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

### Example with bearer token and custom header:
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

## Copyright and License

Copyright (c) 2025 by Tenebris Technologies Inc. and available for use under Apache License 2.0. Please see the LICENSE file for full information.

## Trademarks

Any trademarks referenced are the propery of their respective owners, used for identification only, and do not imply sponsorship, endorsement, or affiliation.

## No Warranty (zilch, none, void, nil, null, "", {}, 0x00, 0b00000000, EOF)

THIS SOFTWARE IS PROVIDED “AS IS,” WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND NON-INFRINGEMENT. IN NO EVENT SHALL THE COPYRIGHT HOLDERS OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

Made in Canada
