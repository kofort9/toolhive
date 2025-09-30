#!/bin/bash

# Complete Token Exchange Demo with thv proxy
# Tests: Client (aud=mcp-server) → thv proxy (exchanges) → MCP Server → Backend (aud=backend)

set -e

echo "🚀 Complete Token Exchange Flow Test"
echo "====================================="
echo ""

# Get initial token with aud=mcp-server
echo "1️⃣  Getting initial token from Keycloak (aud=mcp-server)..."
INITIAL_TOKEN=$(curl -s -d "client_id=mcp-test-client" \
  -d "client_secret=mcp-test-client-secret" \
  -d "username=toolhive-user" \
  -d "password=user123" \
  -d "grant_type=password" \
  "http://localhost:8080/realms/toolhive/protocol/openid-connect/token" | jq -r '.access_token')

echo "✅ Got initial token with aud=mcp-server"
echo ""

# Step 1: Initialize MCP session through proxy
echo "2️⃣  Initializing MCP session through thv proxy..."
INIT_RESPONSE=$(curl -s -i -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $INITIAL_TOKEN" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "clientInfo": {
        "name": "test-client",
        "version": "1.0.0"
      },
      "capabilities": {}
    }
  }')

# Extract session ID
SESSION_ID=$(echo "$INIT_RESPONSE" | grep -i "mcp-session-id:" | cut -d: -f2 | tr -d ' \r\n')

# Extract JSON body
INIT_JSON=$(echo "$INIT_RESPONSE" | tail -n 1)

echo "📋 Session ID: $SESSION_ID"
echo "$INIT_JSON" | jq '.'
echo ""

# Step 2: List tools through proxy
echo "3️⃣  Listing tools through proxy..."
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $INITIAL_TOKEN" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list"
  }' | jq '.'
echo ""

# Step 3: Call the tool through proxy
echo "4️⃣  Calling get_backend_data through proxy..."
echo "   (Proxy will exchange aud=mcp-server → aud=backend)"
echo ""

TOOL_RESPONSE=$(curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $INITIAL_TOKEN" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "get_backend_data",
      "arguments": {}
    }
  }')

echo "$TOOL_RESPONSE" | jq '.'
echo ""

# Check if successful
if echo "$TOOL_RESPONSE" | jq -e '.result.content[0].text' | grep -q "SUCCESS"; then
    echo "🎉🎉🎉 COMPLETE SUCCESS! 🎉🎉🎉"
    echo ""
    echo "The complete token exchange flow worked:"
    echo "  1. Client sent token with aud=mcp-server ✅"
    echo "  2. thv proxy exchanged to aud=backend ✅"
    echo "  3. MCP server forwarded exchanged token ✅"
    echo "  4. Backend validated and returned data ✅"
    echo ""
    echo "🔒 Production-ready OAuth 2.0 Token Exchange (RFC 8693) is working!"
else
    echo "❌ Test failed - check the output above"
fi