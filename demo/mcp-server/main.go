package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Simple MCP Server for Token Exchange Demo
// This server provides one tool that calls the backend service,
// demonstrating how the token exchange middleware works.

func main() {
	backendURL := getEnv("BACKEND_URL", "http://localhost:8090")
	port := getEnv("PORT", "8091")

	// Create MCP server with tool capabilities
	s := server.NewMCPServer(
		"Demo MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Add single tool that calls the backend
	s.AddTool(
		mcp.NewTool("get_backend_data",
			mcp.WithDescription("🎯 Fetches secure data from the backend service (requires aud=backend token)"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetBackendData(ctx, req, backendURL)
		},
	)

	log.Printf("🚀 Demo MCP Server starting on port %s", port)
	log.Printf("🔗 Backend URL: %s", backendURL)
	log.Printf("🎯 This server demonstrates token exchange by forwarding Bearer tokens to the backend")
	log.Printf("📡 Tool available: get_backend_data")

	// Start StreamableHTTP server
	httpServer := server.NewStreamableHTTPServer(s)
	if err := httpServer.Start(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleGetBackendData(ctx context.Context, req mcp.CallToolRequest, backendURL string) (*mcp.CallToolResult, error) {
	// Extract Authorization header from the request
	authHeader := req.Header.Get("Authorization")

	if authHeader == "" {
		log.Printf("⚠️  No Authorization header in request")
		return mcp.NewToolResultText("⚠️  No Authorization header found in request"), nil
	}

	log.Printf("🔑 Received Authorization header, forwarding to backend")

	// Call backend service
	url := backendURL + "/api/data"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("❌ Failed to create request: %v", err)), nil
	}

	// Forward the Authorization header to the backend
	httpReq.Header.Set("Authorization", authHeader)
	log.Printf("🔗 Forwarding to %s with Authorization header", url)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("❌ Backend request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("❌ Failed to read response: %v", err)), nil
	}

	// Pretty print JSON response
	var jsonData interface{}
	var responseText string
	if err := json.Unmarshal(body, &jsonData); err == nil {
		prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
		responseText = string(prettyJSON)
	} else {
		responseText = string(body)
	}

	// Check response status
	if resp.StatusCode == 200 {
		log.Printf("✅ Backend call succeeded (HTTP %d)", resp.StatusCode)
		return mcp.NewToolResultText(fmt.Sprintf("✅ Backend Data Retrieved Successfully!\n\nHTTP %d\n\n%s", resp.StatusCode, responseText)), nil
	}

	log.Printf("❌ Backend call failed (HTTP %d)", resp.StatusCode)
	return mcp.NewToolResultError(fmt.Sprintf("❌ Backend Call Failed\n\nHTTP %d\n\n%s", resp.StatusCode, responseText)), nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}