package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type Config struct {
	Port         string
	KeycloakURL  string
	Realm        string
	JWKSUrl      string
	RequiredAud  string
	RequiredScope string
}

type BackendService struct {
	config *Config
	jwkSet jwk.Set
}

type Claims struct {
	jwt.RegisteredClaims
	Scope            string                 `json:"scope"`
	PreferredUsername string                `json:"preferred_username"`
	Email            string                 `json:"email"`
	Name             string                 `json:"name"`
	RealmAccess      map[string]interface{} `json:"realm_access"`
	AZP              string                 `json:"azp"`
}

func main() {
	config := &Config{
		Port:          getEnv("PORT", "8090"),
		KeycloakURL:   getEnv("KEYCLOAK_URL", "http://localhost:8080"),
		Realm:         getEnv("KEYCLOAK_REALM", "toolhive"),
		RequiredAud:   getEnv("REQUIRED_AUDIENCE", "backend"),
		RequiredScope: getEnv("REQUIRED_SCOPE", "backend-access"),
	}
	config.JWKSUrl = fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", config.KeycloakURL, config.Realm)

	service := &BackendService{config: config}

	// Initialize JWKS
	if err := service.initJWKS(); err != nil {
		log.Fatalf("Failed to initialize JWKS: %v", err)
	}

	// Setup routes
	http.HandleFunc("/health", service.healthHandler)
	http.HandleFunc("/api/data", service.requireAuth(service.dataHandler))
	http.HandleFunc("/api/debug", service.requireAuth(service.debugHandler))

	log.Printf("🚀 Backend Service starting on port %s", config.Port)
	log.Printf("🔒 JWKS URL: %s", config.JWKSUrl)
	log.Printf("🎯 Required Audience: %s", config.RequiredAud)
	log.Printf("📜 Required Scope: %s", config.RequiredScope)
	log.Printf("📡 Endpoints:")
	log.Printf("   GET /health - Health check")
	log.Printf("   GET /api/data - Main endpoint (requires valid JWT)")
	log.Printf("   GET /api/debug - Token debug info")

	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func (s *BackendService) initJWKS() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jwkSet, err := jwk.Fetch(ctx, s.config.JWKSUrl)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	s.jwkSet = jwkSet
	log.Printf("✅ JWKS initialized successfully")
	return nil
}

func (s *BackendService) validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from token header
		kidInterface, ok := token.Header["kid"]
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}
		kid, ok := kidInterface.(string)
		if !ok {
			return nil, fmt.Errorf("kid is not a string")
		}

		// Find key in JWKS
		key, found := s.jwkSet.LookupKeyID(kid)
		if !found {
			return nil, fmt.Errorf("key not found in JWKS")
		}

		var pubKey interface{}
		if err := key.Raw(&pubKey); err != nil {
			return nil, fmt.Errorf("failed to get public key: %w", err)
		}

		return pubKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate audience
	found := false
	for _, aud := range claims.Audience {
		if aud == s.config.RequiredAud {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("invalid audience: expected %s, got %v", s.config.RequiredAud, claims.Audience)
	}

	// Validate scope
	scopes := strings.Split(claims.Scope, " ")
	hasRequiredScope := false
	for _, scope := range scopes {
		if scope == s.config.RequiredScope {
			hasRequiredScope = true
			break
		}
	}
	if !hasRequiredScope {
		return nil, fmt.Errorf("missing required scope: %s", s.config.RequiredScope)
	}

	return claims, nil
}

func (s *BackendService) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"Missing or invalid Authorization header","demo_note":"🚫 No token provided"}`, http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := s.validateToken(token)
		if err != nil {
			log.Printf("❌ Token validation failed: %v", err)
			response := map[string]interface{}{
				"error":     "Token validation failed",
				"details":   err.Error(),
				"demo_note": "🚫 Token validation failed - production-ready validation!",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Log successful validation
		log.Printf("✅ Valid token received at %s", time.Now().Format(time.RFC3339))
		log.Printf("👤 User: %s", claims.PreferredUsername)
		log.Printf("🎯 Audience: %v", claims.Audience)
		log.Printf("📜 Scopes: %s", claims.Scope)
		log.Printf("🏢 Client: %s", claims.AZP)
		log.Println("---")

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "claims", claims)
		next(w, r.WithContext(ctx))
	}
}

func (s *BackendService) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":            "healthy",
		"service":           "demo-backend",
		"jwks_url":          s.config.JWKSUrl,
		"expected_audience": s.config.RequiredAud,
		"required_scope":    s.config.RequiredScope,
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *BackendService) dataHandler(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*Claims)

	response := map[string]interface{}{
		"message": "🎉 SUCCESS! Token exchange worked perfectly!",
		"backend_data": map[string]interface{}{
			"secure_info": "This is sensitive backend data that requires aud=backend",
			"timestamp":   time.Now().Format(time.RFC3339),
			"user_info": map[string]interface{}{
				"subject":  claims.Subject,
				"username": claims.PreferredUsername,
				"email":    claims.Email,
				"name":     claims.Name,
			},
			"token_info": map[string]interface{}{
				"audience": claims.Audience,
				"scopes":   strings.Split(claims.Scope, " "),
				"issuer":   claims.Issuer,
				"client":   claims.AZP,
			},
		},
		"demo_note":     "✅ This proves the token was exchanged from aud=server to aud=backend!",
		"security_note": "🔒 This endpoint uses production-ready JWT validation with JWKS",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *BackendService) debugHandler(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*Claims)

	response := map[string]interface{}{
		"validated_token_payload": map[string]interface{}{
			"subject":            claims.Subject,
			"audience":           claims.Audience,
			"issuer":             claims.Issuer,
			"scope":              claims.Scope,
			"preferred_username": claims.PreferredUsername,
			"email":              claims.Email,
			"name":               claims.Name,
			"azp":                claims.AZP,
			"issued_at":          time.Unix(claims.IssuedAt.Unix(), 0).Format(time.RFC3339),
			"expires_at":         time.Unix(claims.ExpiresAt.Unix(), 0).Format(time.RFC3339),
		},
		"demo_note":     "🔍 This shows the validated JWT payload",
		"security_note": "🔒 Token signature, expiry, audience, and scope were all validated",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}