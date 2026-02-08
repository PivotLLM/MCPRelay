# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MCPRelay is a Go-based proxy that enables MCP clients supporting only stdio transport to connect to network MCP servers using either HTTP or SSE (Server-Sent Events) transport. It bridges the communication gap by:
- Reading JSON-RPC messages from stdin (from MCP client)
- Forwarding them to the MCP server (via HTTP POST or SSE)
- Receiving server responses (via HTTP response or SSE stream)
- Writing responses back to stdout (to MCP client)

**Release Status**: This code is UNRELEASED. Backward compatibility is NOT required for refactoring or changes.

## Build Commands

Build the binary:
```bash
go build -o mcprelay .
```

The build produces a single binary `mcprelay` with no external dependencies beyond Go stdlib.

## Architecture

### Package Structure

- **main.go**: Entry point, parses command-line flags (-url, -log, -debug, -headers, -transport), initializes logger and relay
- **relay/relay.go**: Core relay logic with transport-specific implementations:
  - `Run()`: Routes to transport-specific implementation
  - `runHTTP()`: HTTP mode - synchronous request/response loop
  - `runSSE()`: SSE mode - concurrent goroutines for stdin and SSE stream
  - `processHTTPRequest()`: HTTP mode request handler
  - `processStdinLine()`: SSE mode stdin handler
  - `sseClient()`: Maintains persistent SSE connection, forwards server messages to stdout
- **data/data.go**: Thread-safe data store for server URLs using sync.RWMutex

### Transport Modes

MCPRelay supports two transport modes:

**HTTP Transport (default)**:
- Uses standard HTTP POST for request/response
- Each stdin JSON-RPC message → POST to server → immediate response → stdout
- Synchronous, stateless operation
- Single-threaded processing loop in runHTTP()
- POST endpoint specified via `-url` flag
- Communication flow: Client → stdin → MCPRelay → HTTP POST → Server → HTTP Response → MCPRelay → stdout → Client

**SSE Transport (legacy)**:
- Maintains persistent GET connection for SSE stream
- POST requests don't return responses (responses come via SSE events)
- Asynchronous, requires goroutine coordination
- Supports dynamic endpoint discovery via "event: endpoint" messages
- SSE endpoint specified via `-url` flag
- Communication flow:
  - Upstream: Client → stdin → MCPRelay → HTTP POST → Server
  - Downstream: Server → SSE stream → MCPRelay → stdout → Client

Use `-transport http` (default) or `-transport sse` to select mode.

### Key Design Patterns

- **Transport separation**: HTTP and SSE modes use separate implementations (runHTTP vs runSSE) due to fundamentally different concurrency models
- **Goroutine coordination** (SSE mode only): Uses channels (sseConnected, stdinChan, stdinErrChan) to coordinate stdin reading, SSE connection establishment, and message forwarding
- **Context cancellation** (SSE mode): Clean shutdown when stdin closes using context.Context
- **Thread-safe URL management**: Data package protects concurrent access to server URLs
- **Dynamic endpoint discovery** (SSE mode only): SSE server can send `event: endpoint` followed by `data: /path` to change POST endpoint dynamically
- **Custom headers**: -headers flag allows passing authentication tokens and custom HTTP headers to all requests (both modes)

### Logging

- Logs to file specified by -log flag (required since stdio is used for MCP protocol)
- Debug mode (-debug) adds file:line information and detailed request/response logging
- All logging uses log.Logger, can be disabled with empty -log flag

### Important Implementation Notes

- stdin/stdout is the MCP protocol interface - never write non-protocol messages to stdout
- **HTTP mode**: Simple synchronous loop, no goroutines, immediate responses
- **SSE mode**: SSE connection must establish before processing stdin to avoid race conditions
- **SSE mode**: SSE reconnects automatically with 5-second backoff on disconnection
- Mutex protects stdout writes to prevent message interleaving (both modes)
- Custom headers are applied to all HTTP requests (SSE GET, HTTP POST, message POST)
