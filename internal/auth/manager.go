package auth

import (
	"biblio-ebooks-catalog/internal/db"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// AuthMode represents the authentication mode
type AuthMode string

const (
	AuthModeInternal   AuthMode = "internal"
	AuthModeOIDC       AuthMode = "oidc"
	AuthModeBiblioAuth AuthMode = "biblio-auth"
)

// Manager handles authentication for internal, oidc, and biblio-auth modes
type Manager struct {
	mode         AuthMode
	internalAuth *Auth             // Used in all modes for Basic Auth (OPDS)
	biblioAuth   *BiblioAuthClient // Used in biblio-auth mode for web UI
}

// NewManager creates a new authentication manager
func NewManager(mode string, database *db.DB, biblioAuthURL string) (*Manager, error) {
	m := &Manager{
		mode:         AuthMode(mode),
		internalAuth: New(database),
	}

	// Initialize biblio-auth client if in biblio-auth mode
	if m.mode == AuthModeBiblioAuth {
		m.biblioAuth = NewBiblioAuthClient(biblioAuthURL)
	}

	return m, nil
}

// GetMode returns the current authentication mode
func (m *Manager) GetMode() AuthMode {
	return m.mode
}

// IsInternalMode returns true if using internal authentication
func (m *Manager) IsInternalMode() bool {
	return m.mode == AuthModeInternal
}

// IsBiblioAuthMode returns true if using Biblio Auth
func (m *Manager) IsBiblioAuthMode() bool {
	return m.mode == AuthModeBiblioAuth
}

// GetInternalAuth returns the internal auth provider for Basic Auth
func (m *Manager) GetInternalAuth() *Auth {
	return m.internalAuth
}

// GetBiblioAuth returns the Biblio Auth client
func (m *Manager) GetBiblioAuth() *BiblioAuthClient {
	return m.biblioAuth
}

// Authenticate authenticates a user with username/password (for Basic Auth/OPDS)
func (m *Manager) Authenticate(username, password string) (*db.User, error) {
	return m.internalAuth.Authenticate(username, password)
}

// CreateSession creates a session for a user (for Basic Auth/OPDS)
func (m *Manager) CreateSession(userID int64) (*db.Session, error) {
	return m.internalAuth.CreateSession(userID)
}

// ValidateSession validates a session (for Basic Auth/OPDS)
func (m *Manager) ValidateSession(sessionID string) (*db.User, error) {
	return m.internalAuth.ValidateSession(sessionID)
}

// ValidateBiblioAuthSession validates a Biblio Auth session token
func (m *Manager) ValidateBiblioAuthSession(token string) (*UserInfo, error) {
	if m.biblioAuth == nil {
		return nil, fmt.Errorf("biblio-auth not configured")
	}
	return m.biblioAuth.ValidateSession(token)
}

// GetLoginURL returns the Biblio Auth login URL
func (m *Manager) GetLoginURL(returnURL string) string {
	if m.biblioAuth == nil {
		return ""
	}
	return m.biblioAuth.GetLoginURL(returnURL)
}

// GetLogoutURL returns the Biblio Auth logout URL
func (m *Manager) GetLogoutURL() string {
	if m.biblioAuth == nil {
		return ""
	}
	return m.biblioAuth.GetLogoutURL()
}

// IsBiblioAuthAdmin checks if a Biblio Auth user has admin privileges
func (m *Manager) IsBiblioAuthAdmin(userInfo *UserInfo) bool {
	if m.biblioAuth == nil {
		return false
	}
	return m.biblioAuth.IsAdmin(userInfo)
}

// DeleteSession deletes a session (internal mode only)
func (m *Manager) DeleteSession(sessionID string) error {
	return m.internalAuth.DeleteSession(sessionID)
}

// GetUsers returns all users (internal mode only)
func (m *Manager) GetUsers() ([]db.User, error) {
	return m.internalAuth.GetUsers()
}

// GetUser returns a user by ID (internal mode only)
func (m *Manager) GetUser(id int64) (*db.User, error) {
	return m.internalAuth.GetUser(id)
}

// CreateUser creates a new user (internal mode only)
func (m *Manager) CreateUser(username, password, role string) (*db.User, error) {
	return m.internalAuth.CreateUser(username, password, role)
}

// UpdateUserPassword updates a user's password (internal mode only)
func (m *Manager) UpdateUserPassword(id int64, newPassword string) error {
	return m.internalAuth.UpdateUserPassword(id, newPassword)
}

// UpdateUserRole updates a user's role (internal mode only)
func (m *Manager) UpdateUserRole(id int64, role string) error {
	return m.internalAuth.UpdateUserRole(id, role)
}

// DeleteUser deletes a user (internal mode only)
func (m *Manager) DeleteUser(id int64) error {
	return m.internalAuth.DeleteUser(id)
}

// HasUsers checks if any users exist (internal mode only)
func (m *Manager) HasUsers() (bool, error) {
	return m.internalAuth.HasUsers()
}

// CheckSessionOrBasicAuth checks for session or basic auth
// In biblio-auth mode, validates against Biblio Auth service
// In internal mode, validates against internal database
func (m *Manager) CheckSessionOrBasicAuth(w http.ResponseWriter, r *http.Request) bool {
	// In biblio-auth mode, check auth_token cookie first, then Basic Auth via Biblio Auth
	if m.mode == AuthModeBiblioAuth {
		// Check for auth_token cookie (from Biblio Auth web login)
		cookie, err := r.Cookie("auth_token")
		if err == nil {
			userInfo, err := m.biblioAuth.ValidateSession(cookie.Value)
			if err == nil {
				// Convert to db.User for context
				user := &db.User{
					ID:       int64(userInfo.ID),
					Username: userInfo.Username,
					Role:     "user",
				}
				for _, group := range userInfo.Groups {
					if group == "admin" {
						user.Role = db.RoleAdmin
						break
					}
				}
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				*r = *r.WithContext(ctx)
				return true
			}
		}

		// Try Basic Auth via Biblio Auth
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Basic ") {
			decoded, err := base64.StdEncoding.DecodeString(authHeader[6:])
			if err == nil {
				parts := strings.SplitN(string(decoded), ":", 2)
				if len(parts) == 2 {
					userInfo, err := m.biblioAuth.ValidateBasicAuth(parts[0], parts[1])
					if err == nil {
						log.Printf("Biblio Auth Basic Auth successful: user=%s", userInfo.Username)
						// Convert to db.User for context
						user := &db.User{
							ID:       int64(userInfo.ID),
							Username: userInfo.Username,
							Role:     "user",
						}
						for _, group := range userInfo.Groups {
							if group == "admin" {
								user.Role = db.RoleAdmin
								break
							}
						}
						ctx := context.WithValue(r.Context(), UserContextKey, user)
						*r = *r.WithContext(ctx)
						return true
					}
					log.Printf("Biblio Auth Basic Auth failed: %v", err)
				}
			}
		}

		// Request Basic Auth
		w.Header().Set("WWW-Authenticate", `Basic realm="opds-server"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	// In internal mode, use the standard session/basic auth validation
	return m.internalAuth.CheckSessionOrBasicAuth(w, r)
}

// CheckSession checks for session auth
func (m *Manager) CheckSession(w http.ResponseWriter, r *http.Request) bool {
	// In biblio-auth mode, check for auth_token cookie
	if m.mode == AuthModeBiblioAuth {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Not authenticated. Please log in again."}`))
			return false
		}

		userInfo, err := m.biblioAuth.ValidateSession(cookie.Value)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Session expired. Please log in again."}`))
			return false
		}

		// Convert to db.User for context
		user := &db.User{
			ID:       int64(userInfo.ID),
			Username: userInfo.Username,
			Role:     "user",
		}
		for _, group := range userInfo.Groups {
			if group == "admin" {
				user.Role = db.RoleAdmin
				break
			}
		}
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		*r = *r.WithContext(ctx)
		return true
	}

	// In internal mode, use the standard session validation
	return m.internalAuth.CheckSession(w, r)
}

// CheckAdmin checks if the current user is an admin
func (m *Manager) CheckAdmin(w http.ResponseWriter, r *http.Request) bool {
	return m.internalAuth.CheckAdmin(w, r)
}
