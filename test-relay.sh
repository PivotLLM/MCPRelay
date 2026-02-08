#!/bin/bash

# Test script for MCPRelay using probe
# This tests MCPRelay by running it in stdio mode and using probe to communicate with it

set -e

# Configuration
SERVER_URL="http://10.42.10.75:9999/mcp"
BEARER_TOKEN="c78fe472ef6c9645e9111f663b24ae4ccdfd268c76309d718cafc3c92af2e251"
LOG_FILE="/tmp/relay-test.log"
RELAY_BIN="./mcprelay"

# Check if mcprelay binary exists
if [ ! -f "$RELAY_BIN" ]; then
    echo "Error: mcprelay binary not found. Building..."
    go build -o mcprelay .
fi

# Build arguments for MCPRelay
# Note: probe -args expects comma-separated values
RELAY_ARGS="-url,$SERVER_URL,-headers,{\\\"Authorization\\\":\\\"Bearer $BEARER_TOKEN\\\"},-log,$LOG_FILE,-debug"

echo "========================================"
echo "Testing MCPRelay with probe"
echo "========================================"
echo "Server URL: $SERVER_URL"
echo "Log file: $LOG_FILE"
echo "Relay args: $RELAY_ARGS"
echo ""

# Clear previous log
rm -f "$LOG_FILE"

# Parse command line arguments
case "${1:-list}" in
    list)
        echo "Listing available tools..."
        probe -stdio "$RELAY_BIN" -args "$RELAY_ARGS" -list-only
        ;;

    list-names)
        echo "Listing tool names only..."
        probe -stdio "$RELAY_BIN" -args "$RELAY_ARGS" -list
        ;;

    call)
        if [ -z "$2" ]; then
            echo "Error: tool name required"
            echo "Usage: $0 call <tool-name> [params-json]"
            exit 1
        fi
        TOOL_NAME="$2"
        PARAMS="${3:-{}}"
        echo "Calling tool: $TOOL_NAME"
        echo "Parameters: $PARAMS"
        probe -stdio "$RELAY_BIN" -args "$RELAY_ARGS" -call "$TOOL_NAME" -params "$PARAMS"
        ;;

    interactive)
        echo "Starting interactive mode..."
        echo "You can now call tools interactively."
        probe -stdio "$RELAY_BIN" -args "$RELAY_ARGS" -interactive
        ;;

    test-connection)
        echo "Testing basic connection..."
        probe -stdio "$RELAY_BIN" -args "$RELAY_ARGS" -list
        if [ $? -eq 0 ]; then
            echo ""
            echo "✓ Connection successful!"
        else
            echo ""
            echo "✗ Connection failed!"
            exit 1
        fi
        ;;

    *)
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  list            - List available tools with details (default)"
        echo "  list-names      - List tool names only"
        echo "  call <tool> [params] - Call a specific tool"
        echo "  interactive     - Interactive tool calling mode"
        echo "  test-connection - Quick connection test"
        echo ""
        echo "Examples:"
        echo "  $0 list"
        echo "  $0 call microsoft365_profile_get"
        echo "  $0 call microsoft365_calendar_read_summary '{\"startDate\":\"20250101\"}'"
        echo "  $0 interactive"
        exit 1
        ;;
esac

echo ""
echo "========================================"
echo "Check log file for details: $LOG_FILE"
echo "========================================"
