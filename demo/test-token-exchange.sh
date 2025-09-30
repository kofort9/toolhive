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

curl -s -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "client_id=mcp-server" \
  -d "client_secret=PLOs4j6ti521kb5ZVVwi5GWi9eDYTwq" \
  -d "subject_token=$INITIAL_TOKEN" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=backend-access" \
  "http://localhost:8080/realms/toolhive/protocol/openid-connect/token"
