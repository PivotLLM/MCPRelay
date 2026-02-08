/******************************************************************************
 * Copyright (c) 2025 Tenebris Technologies Inc.                              *
 * See LICENSE for details.                                                   *
 ******************************************************************************/

package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"github.com/PivotLLM/MCPRelay/relay"
)

const PRODUCT = "MCPRelay v0.4.0"

func main() {
	var err error
	var logFile *os.File
	var logger *log.Logger

	// Parse command-line flags
	logFilePath := flag.String("log", "", "Path to the log file (leave empty to disable logging)")
	sseURL := flag.String("url", "http://127.0.0.1:8888/sse", "URL to connect to SSE stream")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	headersJSON := flag.String("headers", "", "Custom HTTP headers as JSON object (e.g., '{\"Authorization\":\"Bearer token\"}')")
	transport := flag.String("transport", "http", "Transport mode: 'http' or 'sse'")
	flag.Parse()

	// Validate transport mode
	if *transport != "http" && *transport != "sse" {
		log.Fatalf("Invalid transport mode: %s (must be 'http' or 'sse')", *transport)
	}

	// Parse custom headers if provided
	var headers map[string]string
	if *headersJSON != "" {
		if err := json.Unmarshal([]byte(*headersJSON), &headers); err != nil {
			log.Fatalf("Failed to parse headers JSON: %s", err)
		}
	}

	// Set the default logger to discard
	log.SetOutput(io.Discard)

	// MCP is using stdio, so if user doesn't specify a log path, discard log events
	if *logFilePath == "" {
		logger = log.New(io.Discard, "", 0)
	} else {

		// Open the log file
		logFile, err = os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file %s: %s", *logFilePath, err)
		}

		// Set the log output to the log file
		lFlags := log.LstdFlags
		if *debugFlag {
			lFlags = log.LstdFlags | log.Lshortfile
		}
		logger = log.New(logFile, "", lFlags)
		logger.Printf("%s started", PRODUCT)

		// Ensure the log file is closed when the program exits
		defer func() {
			if logFile != nil {
				_ = logFile.Close()
			}
		}()
	}

	// Instantiate the relay
	r, err := relay.New(*sseURL, logger, logFile, *debugFlag, headers, *transport)
	if err != nil {
		logger.Fatalf("Failed to create relay: %s", err.Error())
	}

	// Run the relay
	// This will block until the client disconnects
	r.Run()

	// Log exit
	logger.Printf("%s exiting", PRODUCT)
}
