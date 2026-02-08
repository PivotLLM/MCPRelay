/******************************************************************************
 * Copyright (c) 2025 Tenebris Technologies Inc.                              *
 * See LICENSE for details.                                                   *
 ******************************************************************************/

package relay

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PivotLLM/MCPRelay/data"
)

// Logger is an alias for log.Logger
type Logger = *log.Logger

type Relay struct {
	writerMutex sync.Mutex
	debug       bool
	logger      Logger
	logFile     *os.File
	data        *data.Data
	headers     map[string]string
	transport   string       // "http" or "sse"
	httpClient  *http.Client // persistent HTTP client for keep-alive
	sessionID   string       // MCP session ID for HTTP transport
}

func New(endpoint string, logger Logger, logFile *os.File, debug bool, headers map[string]string, transport string) (*Relay, error) {
	var err error

	// Instantiate our object
	r := &Relay{
		logger:    logger,
		logFile:   logFile,
		debug:     debug,
		data:      data.New(logger),
		headers:   headers,
		transport: transport,
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	// Protect against nil logger
	if r.logger == nil {
		r.logger = log.New(io.Discard, "", 0)
	}

	// Set up data store
	r.data = data.New(r.logger)

	// Mode-specific setup
	if transport == "sse" {
		// Parse URL for SSE mode
		var u *url.URL
		u, err = url.Parse(endpoint)
		if err != nil {
			msg := fmt.Sprintf("Error parsing URL '%s': %s", endpoint, err.Error())

			// Advise the MCP client if it is listening
			r.sendClientError(msg)

			// Log fatal error
			return &Relay{}, errors.New(msg)
		}

		// Set the server based on parsing
		// This will avoid repeated parsing if the SSE server responds with a dynamic endpoint
		r.data.SetServer(fmt.Sprintf("%s://%s", u.Scheme, u.Host))

		// Set the SSE URL as specified by the user
		r.data.SetSSEURL(endpoint)

		// Set the default POST endpoint for SSE
		r.data.SetPostPath("/messages")
	} else {
		// HTTP mode: URL is the POST endpoint directly
		r.data.SetPostURL(endpoint)
		r.logger.Printf("HTTP mode: POST endpoint set to %s", endpoint)
	}

	// Return object
	return r, nil
}

// flushLog syncs the log file to disk if one is configured
func (r *Relay) flushLog() {
	if r.logFile != nil {
		_ = r.logFile.Sync()
	}
}

func (r *Relay) Run() {
	if r.transport == "http" {
		r.runHTTP()
	} else {
		r.runSSE()
	}
}

func (r *Relay) runHTTP() {
	r.logger.Println("Starting HTTP mode")
	r.flushLog()

	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				r.logger.Println("EOF on stdin, client closed connection")
			} else {
				r.logger.Printf("stdin error: %s", err.Error())
			}
			r.flushLog()
			return
		}

		// Process message and get response
		response := r.processHTTPRequest(line)
		if response != nil {
			r.sendToClient(response)
		}
	}
}

func (r *Relay) createErrorResponse(requestJSON string, code int, message string) []byte {
	// Try to extract the id from the request
	var id interface{} = nil
	if requestJSON != "" {
		var req map[string]interface{}
		if err := json.Unmarshal([]byte(requestJSON), &req); err == nil {
			if reqID, ok := req["id"]; ok {
				id = reqID
			}
		}
	}

	// Construct proper JSON-RPC 2.0 error response
	errResp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	respBytes, _ := json.Marshal(errResp)
	return respBytes
}

func (r *Relay) processHTTPRequest(line string) []byte {
	// Trim whitespace
	line = strings.TrimSpace(line)

	// Validate JSON-RPC format
	if !strings.HasPrefix(line, "{") {
		r.logger.Printf("Unexpected input: %s", line)
		return nil
	}

	// Parse JSON
	var jsonMsg map[string]interface{}
	if err := json.Unmarshal([]byte(line), &jsonMsg); err != nil {
		r.logger.Printf("Invalid JSON: %s", err.Error())
		return nil
	}

	// Check if this is a notification (no id field)
	_, hasID := jsonMsg["id"]
	isNotification := !hasID

	if r.debug {
		if isNotification {
			r.logger.Println("C->S (notification):", line)
		} else {
			r.logger.Println("C->S:", line)
		}
	}

	// For notifications in HTTP mode, we don't send them to the server
	// The server doesn't support notifications over HTTP transport
	if isNotification {
		if r.debug {
			r.logger.Println("Skipping notification in HTTP mode (not supported by HTTP transport)")
		}
		return nil
	}

	// Build POST request
	postURL := r.data.GetPostURL()
	req, _ := http.NewRequest("POST", postURL, bytes.NewReader([]byte(line)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Add session ID as header if we have one (per MCP spec)
	if r.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", r.sessionID)
	}

	// Add custom headers (authentication, etc.)
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}

	if r.debug && r.sessionID != "" {
		r.logger.Printf("Sending request with session ID header: Mcp-Session-Id: %s", r.sessionID)
	}

	// Send request using persistent client for keep-alive
	resp, err := r.httpClient.Do(req)
	if err != nil {
		msg := fmt.Sprintf("Failed to POST: %s", err.Error())
		r.logger.Println(msg)
		r.flushLog()
		return r.createErrorResponse(line, -32603, msg)
	}
	defer resp.Body.Close()

	// Extract session ID from Mcp-Session-Id header (if present)
	if sessionID := resp.Header.Get("Mcp-Session-Id"); sessionID != "" && r.sessionID == "" {
		// Store the full session ID including the "mcp-session-" prefix
		r.sessionID = sessionID
		r.logger.Printf("Extracted MCP session ID: %s", r.sessionID)
	}

	if r.debug {
		r.logger.Printf("POST %s -> HTTP %d", postURL, resp.StatusCode)
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("Failed to read response: %s", err.Error())
		r.logger.Println(msg)
		r.flushLog()
		return r.createErrorResponse(line, -32603, msg)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("Server returned HTTP %d", resp.StatusCode)
		r.logger.Println(msg)
		if r.debug && len(respBody) > 0 {
			r.logger.Printf("Server error response: %s", string(respBody))
		}
		r.flushLog()
		return r.createErrorResponse(line, -32603, msg)
	}

	return respBody
}

func (r *Relay) runSSE() {
	// Create a cancellable context for clean shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled when Run() exits

	// SSE connection needs to be established first and many SSE servers will provide a dynamic endpoint
	// Use a channel to signal when the SSE connection is established
	sseConnected := make(chan bool, 1)
	go func() {
		r.sseClient(ctx, sseConnected)
	}()

	// Channel for stdin input
	stdinChan := make(chan string)
	stdinErrChan := make(chan error)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				stdinErrChan <- err
				return
			}
			stdinChan <- line
		}
	}()

	// Wait for SSE connection to be established, but also check for stdin closure
	var pendingLine string
	var sseReady bool

	for !sseReady {
		select {
		case <-sseConnected:
			// SSE connected successfully
			sseReady = true
		case err := <-stdinErrChan:
			// stdin closed before SSE connected
			if err == io.EOF {
				r.logger.Println("EOF on stdin before SSE connected, client has closed the connection")
			} else {
				r.logger.Printf("stdin error before SSE connected: %s", err.Error())
			}
			r.flushLog()
			return
		case line := <-stdinChan:
			// Got stdin input before SSE connected, save it for later
			if pendingLine == "" {
				pendingLine = line
				r.logger.Println("Received stdin input before SSE connected, waiting for SSE...")
			}
			// Continue waiting for SSE or more stdin input
		}
	}

	r.logger.Println("Starting receive loop on stdin")
	r.flushLog()

	// Process any pending line
	if pendingLine != "" {
		r.processStdinLine(pendingLine)
	}

	// Main loop: read and forward requests from stdin
	for {
		select {
		case line := <-stdinChan:
			r.processStdinLine(line)
		case err := <-stdinErrChan:
			if err == io.EOF {
				r.logger.Println("EOF on stdin, client has closed the connection")
			} else {
				r.logger.Printf("stdin read error: %s", err.Error())
			}
			r.flushLog()
			return
		}
	}
}

func (r *Relay) processStdinLine(line string) {
	// Trim whitespace and newlines
	line = strings.TrimSpace(line)

	// Check for MCP JSON-RPC message
	if strings.HasPrefix(line, "{") {
		// Attempt to parse as JSON-RPC request
		var jsonMsg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonMsg); err == nil {
			if r.debug {
				r.logger.Println("C->S:", line)
			}

			// Forward the JSON-RPC message from the client to the server
			postURL := r.data.GetPostURL()

			//r.logger.Printf("POSTing JSON-RPC message to server: %s", postURL)

			req, _ := http.NewRequest("POST", postURL, bytes.NewReader([]byte(line)))
			req.Header.Set("Content-Type", "application/json")

			// Add custom headers
			for key, value := range r.headers {
				req.Header.Set(key, value)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				msg := fmt.Sprintf("Failed to forward JSON-RPC message: %s", err.Error())
				r.logger.Println(msg)
				r.flushLog()
				r.sendClientError(msg)
				return
			}

			// Log HTTP response status
			if r.debug {
				r.logger.Printf("POST %s -> HTTP %d", postURL, resp.StatusCode)
			}

			// Close the response body to avoid resource leaks
			_ = resp.Body.Close()

			// Check for non-2xx status codes
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				msg := fmt.Sprintf("Server returned HTTP %d for POST request", resp.StatusCode)
				r.logger.Println(msg)
				r.flushLog()
			}

			/* TODO - in non-SEE mode, the body would have to be parsed, JSON extracted, and forwarded to the client
			   But in SSE mode, the results in the client receiving two responses and getting confused

				// Read the response body and immediately close it
				var respBody []byte
				respBody, err = io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if err != nil {
					msg := fmt.Sprintf("Failed to read response from server: %v", err)
					r.logger.Println(msg)
					r.sendClientError(msg)
					continue
				}

				// Relay the response back to the MCP client
				r.sendToClient(respBody) // let's not do this for SSE because the client will get it from SSE

			*/
			return
		}
	}

	r.logger.Printf("Unexpected input: %s", line)
}

func (r *Relay) sendClientError(msg string) {
	r.sendToClient(r.createErrorResponse("", -32603, fmt.Sprintf("Internal error: %s", msg)))
}

func (r *Relay) sendToClient(msg []byte) {
	var err error

	// Trim whitespace and newlines
	msg = bytes.TrimRight(bytes.TrimRight(msg, "\r\n\t "), "\r\n\t ")

	// Set our mutex to avoid conflicts writing to stdout
	r.writerMutex.Lock()
	defer r.writerMutex.Unlock()

	if r.debug {
		r.logger.Println("S->C:", string(msg))
	}

	// Add a newline to the end of the message
	msg = append(msg, 0x0a)

	_, err = os.Stdout.Write(msg)
	if err != nil {
		r.logger.Printf("Failed to write response body to stdout: %s", err.Error())
	}

	// Flush stdout so that any buffering doesn't delay it
	_ = os.Stdout.Sync()
}

// Connect and maintain an SSE connection to the server
func (r *Relay) sseClient(ctx context.Context, connected chan bool) {
	var err error
	var epTrack int

	// Get the SSE URL
	sseURL := r.data.GetSSEURL()

	// Loop to reconnect if the SSE stream is closed
	for {
		// Check if context is cancelled (client disconnected)
		select {
		case <-ctx.Done():
			r.logger.Println("SSE client shutting down: stdin connection closed")
			r.flushLog()
			return
		default:
			// Continue with connection
		}

		r.logger.Printf("Connecting to SSE stream at %s", sseURL)
		r.flushLog()

		// Reset endpoint tracker
		epTrack = 0

		// Connect to SSE
		req, _ := http.NewRequest("GET", sseURL, nil)
		req = req.WithContext(ctx) // Allow request to be cancelled

		// Add custom headers
		for key, value := range r.headers {
			req.Header.Set(key, value)
		}

		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			// Check if error is due to context cancellation
			if ctx.Err() != nil {
				r.logger.Println("SSE client shutting down: stdin connection closed")
				r.flushLog()
				return
			}
			r.logger.Printf("Failed to connect to SSE: %v", err)
			r.flushLog()

			// Wait before retrying, but check for cancellation
			select {
			case <-ctx.Done():
				r.logger.Println("SSE client shutting down: stdin connection closed")
				r.flushLog()
				return
			case <-time.After(5 * time.Second):
				// Continue to retry
			}
			continue
		}

		// Log HTTP response status
		r.logger.Printf("Connected to SSE stream at %s (HTTP %d)", sseURL, resp.StatusCode)
		r.flushLog()

		// Check for non-2xx status codes
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			r.logger.Printf("Warning: SSE server returned HTTP %d", resp.StatusCode)
			r.flushLog()
			_ = resp.Body.Close()

			// Wait before retrying, but check for cancellation
			select {
			case <-ctx.Done():
				r.logger.Println("SSE client shutting down: stdin connection closed")
				r.flushLog()
				return
			case <-time.After(5 * time.Second):
				// Continue to retry
			}
			continue
		}

		// Signal that the SSE connection is established
		connected <- true

		// Read SSE stream
		reader := bufio.NewReader(resp.Body)
		var line string
		for {
			line, err = reader.ReadString('\n')
			if err != nil {
				r.logger.Printf("SSE stream error: %v", err)
				r.flushLog()
				break
			}

			// Trim whitespace and newlines
			line = strings.TrimSpace(line)

			// Skip empty lines
			if line == "" {
				continue
			}

			// Detect dynamic endpoint event
			if strings.HasPrefix(line, "event: endpoint") {
				epTrack = 1 // pending - next line should a dynamic endpoint
				if r.debug {
					r.logger.Printf("SSE endpoint event received")
				}
				continue
			}

			// Ignore non-data lines
			if !strings.HasPrefix(line, "data:") {
				continue
			}

			// Extract data part
			tmp := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if tmp != "" {
				if r.debug {
					//r.logger.Printf("SSE data: %s", tmp)
				}

				// Is dynamic endpoint pending?
				if epTrack == 1 {
					if strings.HasPrefix(tmp, "/") {

						// Set the dynamic endpoint
						r.data.SetPostPath(tmp)

						// Stop looking until a new SSE session is established
						epTrack = 3
						continue
					} else {
						// Log error, stop looking, but allow data to be forwarded to client
						r.logger.Printf("Expected dynamic endpoint, but recieved: %s", tmp)
						r.flushLog()
						epTrack = 3
					}
				}

				// Forward data to the client
				r.sendToClient([]byte(tmp))
			}
		}

		// Close the response body to avoid resource leaks
		if resp != nil {
			_ = resp.Body.Close()
		}
		r.logger.Println("SSE stream closed, waiting 5 seconds before reconnection attempt")
		r.flushLog()

		// Wait before retrying, but check for cancellation
		select {
		case <-ctx.Done():
			r.logger.Println("SSE client shutting down: stdin connection closed")
			r.flushLog()
			return
		case <-time.After(5 * time.Second):
			// Continue to retry
		}
	}
}
