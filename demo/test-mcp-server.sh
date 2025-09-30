#!/bin/bash

# Test script for MCP server with proper session initialization

set -e

echo "🧪 Testing MCP Server with Token"
echo "================================"
echo ""

# Get exchanged token (aud=backend)
echo "1️⃣  Getting exchanged token from Keycloak..."
INITIAL_TOKEN=$(curl -s -d "client_id=mcp-test-client" \
  -d "client_secret=mcp-test-client-secret" \
  -d "username=toolhive-user" \
  -d "password=user123" \
  -d "grant_type=password" \
  "http://localhost:8080/realms/toolhive/protocol/openid-connect/token" | jq -r '.access_token')

EXCHANGED_TOKEN=$(curl -s -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "client_id=mcp-server" \
  -d "client_secret=PLOs4j6ti521kb5ZVVwi5GWi9eDYTwq" \
  -d "subject_token=$INITIAL_TOKEN" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=backend-access" \
  "http://localhost:8080/realms/toolhive/protocol/openid-connect/token" | jq -r '.access_token')

echo "✅ Got exchanged token (aud=backend)"
echo ""

# Step 1: Initialize MCP session and capture session ID
echo "2️⃣  Initializing MCP session..."
INIT_RESPONSE=$(curl -s -i -X POST http://localhost:8091/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $EXCHANGED_TOKEN" \
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

# Extract session ID from Mcp-Session-Id header
SESSION_ID=$(echo "$INIT_RESPONSE" | grep -i "mcp-session-id:" | cut -d: -f2 | tr -d ' \r\n')

# Extract JSON body
INIT_JSON=$(echo "$INIT_RESPONSE" | tail -n 1)

echo "📋 Session ID: $SESSION_ID"
echo "$INIT_JSON" | jq '.'
echo ""

# Step 2: List available tools with session ID
echo "3️⃣  Listing available tools..."
curl -s -X POST http://localhost:8091/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $EXCHANGED_TOKEN" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list"
  }' | jq '.'
echo ""

# Step 3: Call the get_backend_data tool with session ID
echo "4️⃣  Calling get_backend_data tool..."
TOOL_RESPONSE=$(curl -s -X POST http://localhost:8091/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $EXCHANGED_TOKEN" \
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
    echo "🎉 SUCCESS! MCP server successfully called backend with exchanged token!"
else
    echo "❌ Test failed - check the output above"
fi