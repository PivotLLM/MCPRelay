package relay

import (
	"bufio"
	"bytes"
	//"encoding/hex"
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
	data        *data.Data
}

func New(sseURL string, logger Logger, debug bool) (*Relay, error) {
	var err error

	// Instantiate our object
	r := &Relay{
		logger: logger,
		debug:  debug,
		data:   data.New(logger),
	}

	// Protect against nil logger
	if r.logger == nil {
		r.logger = log.New(io.Discard, "", 0)
	}

	// Set up data store
	r.data = data.New(r.logger)

	// Parse URL
	var u *url.URL
	u, err = url.Parse(sseURL)
	if err != nil {
		msg := fmt.Sprintf("Error parsing URL '%s': %s", sseURL, err.Error())

		// Advise the MCP client if it is listening
		r.sendClientError(msg)

		// Log fatal error
		return &Relay{}, errors.New(msg)
	}

	// Set the server based on parsing
	// This will avoid repeated parsing if the SSE server responds with a dynamic endpoint
	r.data.SetServer(fmt.Sprintf("%s://%s", u.Scheme, u.Host))

	// Set the SSE URL as specified by the user
	r.data.SetSSEURL(sseURL)

	// Set the default POST endpoint for SSE
	r.data.SetPostPath("/messages")

	// Return object
	return r, nil
}

func (r *Relay) Run() {
	var err error
	var line string

	// SSE connection needs to be established first and many SSE servers will provide a dynamic endpoint
	// Use a channel to signal when the SSE connection is established
	sseConnected := make(chan bool)
	go func() {
		r.sseClient(sseConnected)
	}()

	// Wait for SSE connection to be established
	<-sseConnected

	// Loop, read, and forward requests from stdin
	r.logger.Println("Starting receive loop on stdin")
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				r.logger.Println("EOF on stdin, client has closed the connection")
				return
			}
			r.logger.Printf("Header read error: %s", err.Error())
			continue
		}

		// Log the bytes in hex
		//r.logger.Println("Hex from client:\n" + hex.Dump([]byte(line)))

		// Trim whitespace and newlines
		line = strings.TrimSpace(line)

		// Check for MCP JSON-RPC message
		if strings.HasPrefix(line, "{") {

			// Attempt to parse as JSON-RPC request
			var jsonMsg map[string]interface{}
			if err = json.Unmarshal([]byte(line), &jsonMsg); err == nil {
				if r.debug {
					r.logger.Println("C->S:", line)
				}

				// Forward the JSON-RPC message from the client to the server
				postURL := r.data.GetPostURL()

				//r.logger.Printf("POSTing JSON-RPC message to server: %s", postURL)

				req, _ := http.NewRequest("POST", postURL, bytes.NewReader([]byte(line)))
				req.Header.Set("Content-Type", "application/json")

				//var resp *http.Response
				_, err = http.DefaultClient.Do(req)
				if err != nil {
					msg := fmt.Sprintf("Failed to forward JSON-RPC message: %s", err.Error())
					r.logger.Println(msg)
					r.sendClientError(msg)
					continue
				}

				/*

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

					if r.debug {
						r.logger.Printf("Server response: %s", string(respBody))
					}

					// Relay the response back to the MCP client
					r.sendToClient(respBody) // let's not do this for SSE because the client will get it from SSE

				*/
				continue
			}
		}

		r.logger.Printf("Unexpected input: %s", line)
		continue
	}
}

func (r *Relay) sendClientError(msg string) {
	r.sendToClient([]byte(fmt.Sprintf(`{"error":{"code":-32603,"message":"Internal error: %s"}}`, msg)))
}

func (r *Relay) sendToClient(msg []byte) {
	var err error

	// Trim whitespace and newlines
	msg = bytes.TrimRight(bytes.TrimRight(msg, "\r\n\t "), "\r\n\t ")

	// Log the bytes in hex
	//r.logger.Println("Hex to client:\n" + hex.Dump(msg))

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
func (r *Relay) sseClient(connected chan bool) {
	var err error
	var epTrack int

	// Get the SSE URL
	sseURL := r.data.GetSSEURL()

	// Loop to reconnect if the SSE stream is closed
	for {
		r.logger.Printf("Connecting to SSE stream at %s", sseURL)

		// Reset endpoint tracker
		epTrack = 0

		// Connect to SSE
		req, _ := http.NewRequest("GET", sseURL, nil)
		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			r.logger.Printf("Failed to connect to SSE: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}

		if r.debug {
			r.logger.Printf("Connected to SSE stream at %s", sseURL)
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
		time.Sleep(5 * time.Second)
	}
}
