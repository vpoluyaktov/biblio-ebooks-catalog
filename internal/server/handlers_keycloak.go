package server

import (
	"fmt"
	"net/http"
	"time"

	"biblio-opds-server/internal/auth"
)

// handleKeycloakLogin initiates the Keycloak OAuth2 login flow
func (s *Server) handleKeycloakLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authManager.IsKeycloakMode() {
		s.jsonError(w, "Keycloak login not available in internal auth mode", http.StatusBadRequest)
		return
	}

	loginURL, state, err := s.authManager.GetLoginURL()
	if err != nil {
		s.jsonError(w, fmt.Sprintf("Failed to generate login URL: %v", err), http.StatusInternalServerError)
		return
	}

	// Store state in a temporary cookie for CSRF protection
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	// Return the login URL for the client to redirect to
	s.jsonResponse(w, map[string]string{
		"login_url": loginURL,
	})
}

// handleKeycloakCallback handles the OAuth2 callback from Keycloak
func (s *Server) handleKeycloakCallback(w http.ResponseWriter, r *http.Request) {
	if !s.authManager.IsKeycloakMode() {
		http.Error(w, "Keycloak callback not available in internal auth mode", http.StatusBadRequest)
		return
	}

	// Get state from cookie
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "Missing state cookie", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Get code and state from query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Verify state matches
	if state != stateCookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Exchange code for tokens and get user info
	user, err := s.authManager.HandleCallback(code, state)
	if err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Get the Keycloak provider to access tokens
	kcProvider := s.authManager.GetKeycloakAuth()
	if kcProvider == nil {
		http.Error(w, "Keycloak provider not available", http.StatusInternalServerError)
		return
	}

	// Create a Keycloak session
	// Note: In a real implementation, we'd store the actual OAuth2 tokens
	// For now, we'll create a simplified session
	session := &auth.KeycloakSession{
		IDToken:      "placeholder", // Would be the actual ID token
		AccessToken:  "placeholder", // Would be the actual access token
		RefreshToken: "placeholder", // Would be the actual refresh token
		ExpiresAt:    time.Now().Add(8 * time.Hour),
		Username:     user.Username,
		Role:         user.Role,
	}

	sessionJSON, err := session.ToJSON()
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "keycloak_session",
		Value:    sessionJSON,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	// Redirect to the main page
	basePath := s.config.Server.BasePath
	if basePath == "" {
		basePath = "/"
	}
	http.Redirect(w, r, basePath+"/", http.StatusFound)
}

// handleKeycloakLogout handles logout for Keycloak mode
func (s *Server) handleKeycloakLogout(w http.ResponseWriter, r *http.Request) {
	if !s.authManager.IsKeycloakMode() {
		s.jsonError(w, "Keycloak logout not available in internal auth mode", http.StatusBadRequest)
		return
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "keycloak_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Get the Keycloak logout URL
	basePath := s.config.Server.BasePath
	if basePath == "" {
		basePath = "/"
	}
	redirectURL := fmt.Sprintf("%s%s/", r.Host, basePath)
	logoutURL := s.authManager.GetLogoutURL(redirectURL)

	// Return the logout URL for the client to redirect to
	s.jsonResponse(w, map[string]interface{}{
		"success":    true,
		"logout_url": logoutURL,
	})
}

// handleAuthInfo returns information about the current auth mode
func (s *Server) handleAuthInfo(w http.ResponseWriter, r *http.Request) {
	mode := s.authManager.GetMode()

	response := map[string]interface{}{
		"mode": string(mode),
	}

	// Check if user is authenticated
	if mode == auth.AuthModeInternal {
		cookie, err := r.Cookie("session")
		if err == nil {
			user, err := s.authManager.ValidateSession(cookie.Value)
			if err == nil {
				response["authenticated"] = true
				response["user"] = map[string]interface{}{
					"username": user.Username,
					"role":     user.Role,
				}
			} else {
				response["authenticated"] = false
			}
		} else {
			response["authenticated"] = false
		}
	} else if mode == auth.AuthModeKeycloak {
		cookie, err := r.Cookie("keycloak_session")
		if err == nil {
			session, err := auth.KeycloakSessionFromJSON(cookie.Value)
			if err == nil && session.ExpiresAt.After(time.Now()) {
				response["authenticated"] = true
				response["user"] = map[string]interface{}{
					"username": session.Username,
					"role":     session.Role,
				}
			} else {
				response["authenticated"] = false
			}
		} else {
			response["authenticated"] = false
		}
	}

	s.jsonResponse(w, response)
}
