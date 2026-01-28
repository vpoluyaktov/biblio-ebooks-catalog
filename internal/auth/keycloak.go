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
	ErrKeycloakNotConfigured = errors.New("keycloak not configured")
	ErrInvalidState          = errors.New("invalid state parameter")
	ErrNoIDToken             = errors.New("no id_token in response")
)

// KeycloakProvider implements authentication using Keycloak OIDC
type KeycloakProvider struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
	states       map[string]time.Time // state -> expiry time
}

// KeycloakConfig holds Keycloak configuration
type KeycloakConfig struct {
	URL          string
	Realm        string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// NewKeycloakProvider creates a new Keycloak authentication provider
func NewKeycloakProvider(cfg KeycloakConfig) (*KeycloakProvider, error) {
	if cfg.URL == "" || cfg.Realm == "" || cfg.ClientID == "" {
		return nil, ErrKeycloakNotConfigured
	}

	ctx := context.Background()
	issuerURL := fmt.Sprintf("%s/realms/%s", cfg.URL, cfg.Realm)

	provider, err := oidc.NewProvider(ctx, issuerURL)
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

	kp := &KeycloakProvider{
		provider:     provider,
		verifier:     verifier,
		oauth2Config: oauth2Config,
		states:       make(map[string]time.Time),
	}

	// Start cleanup goroutine for expired states
	go kp.cleanupStates()

	return kp, nil
}

// GetLoginURL generates the Keycloak login URL with state parameter
func (kp *KeycloakProvider) GetLoginURL() (string, string, error) {
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
func (kp *KeycloakProvider) HandleCallback(code, state string) (*db.User, string, string, string, error) {
	// Verify state
	expiry, exists := kp.states[state]
	if !exists {
		return nil, "", "", "", ErrInvalidState
	}
	if time.Now().After(expiry) {
		delete(kp.states, state)
		return nil, "", "", "", ErrInvalidState
	}
	delete(kp.states, state)

	ctx := context.Background()

	// Exchange code for token
	oauth2Token, err := kp.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("failed to exchange token: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, "", "", "", ErrNoIDToken
	}

	// Verify ID token
	idToken, err := kp.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("failed to verify ID token: %w", err)
	}

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

	// Determine role (check if user has 'admin' role in Keycloak)
	role := db.RoleReadonly
	for _, r := range claims.RealmAccess.Roles {
		if r == "admin" {
			role = db.RoleAdmin
			break
		}
	}

	// Create user object (not stored in DB for Keycloak mode)
	user := &db.User{
		ID:       0, // Keycloak users don't have local DB IDs
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

// ValidateToken validates a Keycloak token and returns user info
func (kp *KeycloakProvider) ValidateToken(tokenString string) (*db.User, error) {
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

// GetUserInfo retrieves user information from Keycloak userinfo endpoint
func (kp *KeycloakProvider) GetUserInfo(accessToken string) (map[string]interface{}, error) {
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

// GetLogoutURL returns the Keycloak logout URL
func (kp *KeycloakProvider) GetLogoutURL(redirectURL string) string {
	return fmt.Sprintf("%s/protocol/openid-connect/logout?redirect_uri=%s",
		kp.oauth2Config.Endpoint.AuthURL[:len(kp.oauth2Config.Endpoint.AuthURL)-5], // Remove "/auth" suffix
		redirectURL)
}

// cleanupStates periodically removes expired state parameters
func (kp *KeycloakProvider) cleanupStates() {
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

// KeycloakSession represents a session with Keycloak tokens
type KeycloakSession struct {
	IDToken      string    `json:"id_token"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
}

// ToJSON serializes the session to JSON for cookie storage
func (s *KeycloakSession) ToJSON() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// FromJSON deserializes a session from JSON
func KeycloakSessionFromJSON(data string) (*KeycloakSession, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	var session KeycloakSession
	if err := json.Unmarshal(decoded, &session); err != nil {
		return nil, err
	}

	return &session, nil
}
