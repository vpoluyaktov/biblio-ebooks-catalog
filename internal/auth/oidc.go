package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"biblio-opds-server/internal/db"
)

var (
	ErrOIDCNotConfigured = errors.New("OIDC provider not configured")
	ErrInvalidState      = errors.New("invalid state parameter")
	ErrNoIDToken         = errors.New("no id_token in response")
)

// OIDCProvider implements authentication using OIDC
type OIDCProvider struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
	states       map[string]time.Time // state -> expiry time
}

// OIDCConfig holds OIDC provider configuration
type OIDCConfig struct {
	URL          string
	Realm        string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// NewOIDCProvider creates a new OIDC authentication provider
func NewOIDCProvider(cfg OIDCConfig) (*OIDCProvider, error) {
	if cfg.URL == "" || cfg.Realm == "" || cfg.ClientID == "" {
		return nil, ErrOIDCNotConfigured
	}

	ctx := context.Background()
	issuerURL := fmt.Sprintf("%s/realms/%s", cfg.URL, cfg.Realm)

	// Retry connecting to Keycloak with exponential backoff
	var provider *oidc.Provider
	var err error
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		provider, err = oidc.NewProvider(ctx, issuerURL)
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			waitTime := time.Duration(1<<uint(i)) * time.Second // 1, 2, 4, 8, 16... seconds
			if waitTime > 30*time.Second {
				waitTime = 30 * time.Second
			}
			fmt.Printf("Waiting for OIDC provider to be ready (attempt %d/%d, retrying in %v): %v\n", i+1, maxRetries, waitTime, err)
			time.Sleep(waitTime)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "roles"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})

	kp := &OIDCProvider{
		provider:     provider,
		verifier:     verifier,
		oauth2Config: oauth2Config,
		states:       make(map[string]time.Time),
	}

	// Start cleanup goroutine for expired states
	go kp.cleanupStates()

	return kp, nil
}

// GetLoginURL generates the OIDC login URL with state parameter
func (kp *OIDCProvider) GetLoginURL() (string, string, error) {
	state, err := generateState()
	if err != nil {
		return "", "", err
	}

	// Store state with 10 minute expiry
	kp.states[state] = time.Now().Add(10 * time.Minute)

	url := kp.oauth2Config.AuthCodeURL(state)
	return url, state, nil
}

// HandleCallback processes the OAuth2 callback and returns user info and tokens
func (kp *OIDCProvider) HandleCallback(code, state string) (*db.User, string, string, string, error) {
	fmt.Printf("[DEBUG] OIDCProvider.HandleCallback called with state: %s\n", state)
	fmt.Printf("[DEBUG] Known states: %d\n", len(kp.states))

	// Verify state
	expiry, exists := kp.states[state]
	if !exists {
		fmt.Printf("[ERROR] State not found in known states map\n")
		return nil, "", "", "", ErrInvalidState
	}
	if time.Now().After(expiry) {
		fmt.Printf("[ERROR] State expired at %v\n", expiry)
		delete(kp.states, state)
		return nil, "", "", "", ErrInvalidState
	}
	delete(kp.states, state)
	fmt.Printf("[DEBUG] State validated successfully\n")

	ctx := context.Background()

	fmt.Printf("[DEBUG] Exchanging code for token with OIDC provider...\n")
	// Exchange code for token
	oauth2Token, err := kp.oauth2Config.Exchange(ctx, code)
	if err != nil {
		fmt.Printf("[ERROR] Token exchange failed: %v\n", err)
		return nil, "", "", "", fmt.Errorf("failed to exchange token: %w", err)
	}
	fmt.Printf("[DEBUG] Token exchange successful\n")

	// Extract ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		fmt.Printf("[ERROR] No id_token in response\n")
		return nil, "", "", "", ErrNoIDToken
	}
	fmt.Printf("[DEBUG] ID token extracted, length: %d\n", len(rawIDToken))

	// Verify ID token
	idToken, err := kp.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		fmt.Printf("[ERROR] ID token verification failed: %v\n", err)
		return nil, "", "", "", fmt.Errorf("failed to verify ID token: %w", err)
	}
	fmt.Printf("[DEBUG] ID token verified successfully\n")

	// Extract claims
	var claims struct {
		Sub               string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		RealmAccess       struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, "", "", "", fmt.Errorf("failed to parse claims: %w", err)
	}

	// Determine role (check if user has 'admin' role)
	role := db.RoleReadonly
	for _, r := range claims.RealmAccess.Roles {
		if r == "admin" {
			role = db.RoleAdmin
			break
		}
	}

	// Create user object (not stored in DB for OIDC mode)
	user := &db.User{
		ID:       0, // OIDC users don't have local DB IDs
		Username: claims.PreferredUsername,
		Role:     role,
	}

	// Get refresh token if available
	refreshToken := ""
	if oauth2Token.RefreshToken != "" {
		refreshToken = oauth2Token.RefreshToken
	}

	return user, rawIDToken, oauth2Token.AccessToken, refreshToken, nil
}

// ValidateToken validates an OIDC token and returns user info
func (kp *OIDCProvider) ValidateToken(tokenString string) (*db.User, error) {
	ctx := context.Background()

	idToken, err := kp.verifier.Verify(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	var claims struct {
		Sub               string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		RealmAccess       struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	role := db.RoleReadonly
	for _, r := range claims.RealmAccess.Roles {
		if r == "admin" {
			role = db.RoleAdmin
			break
		}
	}

	user := &db.User{
		ID:       0,
		Username: claims.PreferredUsername,
		Role:     role,
	}

	return user, nil
}

// GetUserInfo retrieves user information from OIDC userinfo endpoint
func (kp *OIDCProvider) GetUserInfo(accessToken string) (map[string]interface{}, error) {
	ctx := context.Background()

	userInfo, err := kp.provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var claims map[string]interface{}
	if err := userInfo.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return claims, nil
}

// GetLogoutURL returns the OIDC logout URL
func (kp *OIDCProvider) GetLogoutURL(redirectURL string) string {
	return fmt.Sprintf("%s/protocol/openid-connect/logout?redirect_uri=%s",
		kp.oauth2Config.Endpoint.AuthURL[:len(kp.oauth2Config.Endpoint.AuthURL)-5], // Remove "/auth" suffix
		redirectURL)
}

// AuthenticateWithPassword authenticates a user using Resource Owner Password Credentials (ROPC) grant.
// This is used for Basic Auth on OPDS feeds when running in OIDC mode.
// The user must have the 'opds_user' role to access OPDS feeds.
func (kp *OIDCProvider) AuthenticateWithPassword(username, password string) (*db.User, error) {
	ctx := context.Background()

	// Use ROPC grant to get token
	token, err := kp.oauth2Config.PasswordCredentialsToken(ctx, username, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, ErrNoIDToken
	}

	// Verify ID token
	idToken, err := kp.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims struct {
		Sub               string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		RealmAccess       struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Check if user has opds_user role (required for OPDS feed access)
	hasOPDSAccess := false
	for _, r := range claims.RealmAccess.Roles {
		if r == "opds_user" {
			hasOPDSAccess = true
			break
		}
	}
	if !hasOPDSAccess {
		return nil, fmt.Errorf("user does not have opds_user role")
	}

	// Determine role (check if user has 'admin' role)
	role := db.RoleReadonly
	for _, r := range claims.RealmAccess.Roles {
		if r == "admin" {
			role = db.RoleAdmin
			break
		}
	}

	user := &db.User{
		ID:       0, // OIDC users don't have local DB IDs
		Username: claims.PreferredUsername,
		Role:     role,
	}

	return user, nil
}

// cleanupStates periodically removes expired state parameters
func (kp *OIDCProvider) cleanupStates() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for state, expiry := range kp.states {
			if now.After(expiry) {
				delete(kp.states, state)
			}
		}
	}
}

// generateState creates a random state parameter for OAuth2
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// OIDCSession represents a session with user info (tokens stored server-side if needed)
type OIDCSession struct {
	ExpiresAt time.Time `json:"expires_at"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
}

// ToJSON serializes the session to JSON for cookie storage
func (s *OIDCSession) ToJSON() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// FromJSON deserializes a session from JSON
func OIDCSessionFromJSON(data string) (*OIDCSession, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	var session OIDCSession
	if err := json.Unmarshal(decoded, &session); err != nil {
		return nil, err
	}

	return &session, nil
}
