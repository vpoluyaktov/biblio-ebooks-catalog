package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"biblio-catalog/internal/auth"
)

// handleOIDCLogin initiates the OIDC OAuth2 login flow
func (s *Server) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authManager.IsOIDCMode() {
		s.jsonError(w, "OIDC login not available in internal auth mode", http.StatusBadRequest)
		return
	}

	loginURL, state, err := s.authManager.GetLoginURL()
	if err != nil {
		s.jsonError(w, fmt.Sprintf("Failed to generate login URL: %v", err), http.StatusInternalServerError)
		return
	}

	// Store state in a temporary cookie for CSRF protection
	// Note: State is also stored in the OIDCProvider's in-memory map
	basePath := s.config.Server.BasePath
	if basePath == "" {
		basePath = "/"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     basePath,
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

// handleOIDCCallback handles the OAuth2 callback from OIDC provider
func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] OIDC callback received: %s", r.URL.String())

	if !s.authManager.IsOIDCMode() {
		log.Printf("[DEBUG] Callback rejected: not in OIDC mode")
		http.Error(w, "OIDC callback not available in internal auth mode", http.StatusBadRequest)
		return
	}

	// Get code and state from query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	log.Printf("[DEBUG] Callback params - code: %s..., state: %s", code[:min(10, len(code))], state)

	if code == "" {
		log.Printf("[DEBUG] Callback rejected: missing authorization code")
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	if state == "" {
		log.Printf("[DEBUG] Callback rejected: missing state parameter")
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	// Clear state cookie if it exists
	basePath := s.config.Server.BasePath
	if basePath == "" {
		basePath = "/"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     basePath,
		MaxAge:   -1,
		HttpOnly: true,
	})

	log.Printf("[DEBUG] Calling HandleCallback to exchange code for tokens...")

	// Exchange code for tokens and get user info
	user, idToken, accessToken, refreshToken, err := s.authManager.HandleCallback(code, state)
	if err != nil {
		log.Printf("[ERROR] HandleCallback failed: %v", err)
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	log.Printf("[DEBUG] HandleCallback succeeded - user: %s, role: %s", user.Username, user.Role)

	// Create an OIDC session with user info only (no tokens - they're too large for cookies)
	// Tokens were validated at login time, so we trust the session data
	_ = idToken      // Tokens validated during callback, not stored
	_ = accessToken  // Could be stored server-side if needed for API calls
	_ = refreshToken // Could be stored server-side if needed for token refresh
	session := &auth.OIDCSession{
		ExpiresAt: time.Now().Add(8 * time.Hour),
		Username:  user.Username,
		Role:      user.Role,
	}

	sessionJSON, err := session.ToJSON()
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oidc_session",
		Value:    sessionJSON,
		Path:     basePath,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	// Redirect to the main page
	http.Redirect(w, r, basePath+"/", http.StatusFound)
}

// handleOIDCLogout handles logout for OIDC mode
func (s *Server) handleOIDCLogout(w http.ResponseWriter, r *http.Request) {
	if !s.authManager.IsOIDCMode() {
		s.jsonError(w, "OIDC logout not available in internal auth mode", http.StatusBadRequest)
		return
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oidc_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Get the OIDC logout URL
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
	} else if mode == auth.AuthModeOIDC {
		cookie, err := r.Cookie("oidc_session")
		if err == nil {
			session, err := auth.OIDCSessionFromJSON(cookie.Value)
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
