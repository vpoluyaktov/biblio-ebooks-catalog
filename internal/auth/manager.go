package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"biblio-opds-server/internal/db"
)

// AuthMode represents the authentication mode
type AuthMode string

const (
	AuthModeInternal AuthMode = "internal"
	AuthModeKeycloak AuthMode = "keycloak"
)

// Manager handles authentication for both internal and Keycloak modes
type Manager struct {
	mode         AuthMode
	internalAuth *Auth
	keycloakAuth *KeycloakProvider
}

// NewManager creates a new authentication manager
func NewManager(mode string, database *db.DB, keycloakCfg KeycloakConfig) (*Manager, error) {
	m := &Manager{
		mode: AuthMode(mode),
	}

	switch m.mode {
	case AuthModeInternal:
		m.internalAuth = New(database)
	case AuthModeKeycloak:
		kc, err := NewKeycloakProvider(keycloakCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Keycloak: %w", err)
		}
		m.keycloakAuth = kc
	default:
		return nil, fmt.Errorf("invalid auth mode: %s (must be 'internal' or 'keycloak')", mode)
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

// IsKeycloakMode returns true if using Keycloak authentication
func (m *Manager) IsKeycloakMode() bool {
	return m.mode == AuthModeKeycloak
}

// GetInternalAuth returns the internal auth provider (only in internal mode)
func (m *Manager) GetInternalAuth() *Auth {
	return m.internalAuth
}

// GetKeycloakAuth returns the Keycloak auth provider (only in Keycloak mode)
func (m *Manager) GetKeycloakAuth() *KeycloakProvider {
	return m.keycloakAuth
}

// Authenticate authenticates a user with username/password (internal mode only)
func (m *Manager) Authenticate(username, password string) (*db.User, error) {
	if m.mode != AuthModeInternal {
		return nil, fmt.Errorf("authenticate not supported in %s mode", m.mode)
	}
	return m.internalAuth.Authenticate(username, password)
}

// CreateSession creates a session for a user (internal mode only)
func (m *Manager) CreateSession(userID int64) (*db.Session, error) {
	if m.mode != AuthModeInternal {
		return nil, fmt.Errorf("create session not supported in %s mode", m.mode)
	}
	return m.internalAuth.CreateSession(userID)
}

// ValidateSession validates a session (internal mode only)
func (m *Manager) ValidateSession(sessionID string) (*db.User, error) {
	if m.mode != AuthModeInternal {
		return nil, fmt.Errorf("validate session not supported in %s mode", m.mode)
	}
	return m.internalAuth.ValidateSession(sessionID)
}

// DeleteSession deletes a session (internal mode only)
func (m *Manager) DeleteSession(sessionID string) error {
	if m.mode != AuthModeInternal {
		return fmt.Errorf("delete session not supported in %s mode", m.mode)
	}
	return m.internalAuth.DeleteSession(sessionID)
}

// HasUsers checks if there are any users (internal mode only)
func (m *Manager) HasUsers() (bool, error) {
	if m.mode != AuthModeInternal {
		return true, nil // In Keycloak mode, users are managed in Keycloak
	}
	return m.internalAuth.HasUsers()
}

// CreateUser creates a new user (internal mode only)
func (m *Manager) CreateUser(username, password, role string) (*db.User, error) {
	if m.mode != AuthModeInternal {
		return nil, fmt.Errorf("create user not supported in %s mode - manage users in Keycloak", m.mode)
	}
	return m.internalAuth.CreateUser(username, password, role)
}

// GetUser gets a user by ID (internal mode only)
func (m *Manager) GetUser(id int64) (*db.User, error) {
	if m.mode != AuthModeInternal {
		return nil, fmt.Errorf("get user not supported in %s mode", m.mode)
	}
	return m.internalAuth.GetUser(id)
}

// GetUsers gets all users (internal mode only)
func (m *Manager) GetUsers() ([]db.User, error) {
	if m.mode != AuthModeInternal {
		return nil, fmt.Errorf("get users not supported in %s mode - manage users in Keycloak", m.mode)
	}
	return m.internalAuth.GetUsers()
}

// UpdateUserPassword updates a user's password (internal mode only)
func (m *Manager) UpdateUserPassword(userID int64, newPassword string) error {
	if m.mode != AuthModeInternal {
		return fmt.Errorf("update password not supported in %s mode - manage users in Keycloak", m.mode)
	}
	return m.internalAuth.UpdateUserPassword(userID, newPassword)
}

// UpdateUserRole updates a user's role (internal mode only)
func (m *Manager) UpdateUserRole(userID int64, role string) error {
	if m.mode != AuthModeInternal {
		return fmt.Errorf("update role not supported in %s mode - manage users in Keycloak", m.mode)
	}
	return m.internalAuth.UpdateUserRole(userID, role)
}

// DeleteUser deletes a user (internal mode only)
func (m *Manager) DeleteUser(userID int64) error {
	if m.mode != AuthModeInternal {
		return fmt.Errorf("delete user not supported in %s mode - manage users in Keycloak", m.mode)
	}
	return m.internalAuth.DeleteUser(userID)
}

// GetLoginURL returns the login URL (Keycloak mode only)
func (m *Manager) GetLoginURL() (string, string, error) {
	if m.mode != AuthModeKeycloak {
		return "", "", fmt.Errorf("get login URL not supported in %s mode", m.mode)
	}
	return m.keycloakAuth.GetLoginURL()
}

// HandleCallback handles OAuth2 callback (Keycloak mode only)
func (m *Manager) HandleCallback(code, state string) (*db.User, error) {
	if m.mode != AuthModeKeycloak {
		return nil, fmt.Errorf("handle callback not supported in %s mode", m.mode)
	}
	return m.keycloakAuth.HandleCallback(code, state)
}

// ValidateToken validates a Keycloak token (Keycloak mode only)
func (m *Manager) ValidateToken(token string) (*db.User, error) {
	if m.mode != AuthModeKeycloak {
		return nil, fmt.Errorf("validate token not supported in %s mode", m.mode)
	}
	return m.keycloakAuth.ValidateToken(token)
}

// GetLogoutURL returns the logout URL (Keycloak mode only)
func (m *Manager) GetLogoutURL(redirectURL string) string {
	if m.mode != AuthModeKeycloak {
		return ""
	}
	return m.keycloakAuth.GetLogoutURL(redirectURL)
}

// CheckSessionOrBasicAuth checks for session or basic auth (works in both modes)
func (m *Manager) CheckSessionOrBasicAuth(w http.ResponseWriter, r *http.Request) bool {
	if m.mode == AuthModeInternal {
		return m.internalAuth.CheckSessionOrBasicAuth(w, r)
	}

	// In Keycloak mode, check for Keycloak session cookie
	cookie, err := r.Cookie("keycloak_session")
	if err == nil {
		session, err := KeycloakSessionFromJSON(cookie.Value)
		if err == nil && session.ExpiresAt.After(time.Now()) {
			// Validate the ID token
			user, err := m.keycloakAuth.ValidateToken(session.IDToken)
			if err == nil {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				*r = *r.WithContext(ctx)
				return true
			}
		}
	}

	// For OPDS e-readers, we can't support Keycloak in this mode
	// Return 401 to prompt for authentication
	w.Header().Set("WWW-Authenticate", `Basic realm="opds-server"`)
	http.Error(w, "Keycloak authentication required. Please use web interface to login.", http.StatusUnauthorized)
	return false
}

// CheckSession checks for session auth (works in both modes)
func (m *Manager) CheckSession(w http.ResponseWriter, r *http.Request) bool {
	if m.mode == AuthModeInternal {
		return m.internalAuth.CheckSession(w, r)
	}

	// In Keycloak mode, check for Keycloak session cookie
	cookie, err := r.Cookie("keycloak_session")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Not authenticated. Please log in."}`))
		return false
	}

	session, err := KeycloakSessionFromJSON(cookie.Value)
	if err != nil || session.ExpiresAt.Before(time.Now()) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Session expired. Please log in again."}`))
		return false
	}

	// Validate the ID token
	user, err := m.keycloakAuth.ValidateToken(session.IDToken)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Invalid session. Please log in again."}`))
		return false
	}

	ctx := context.WithValue(r.Context(), UserContextKey, user)
	*r = *r.WithContext(ctx)
	return true
}

// CheckAdmin checks if the current user is an admin (works in both modes)
func (m *Manager) CheckAdmin(w http.ResponseWriter, r *http.Request) bool {
	if m.mode == AuthModeInternal {
		return m.internalAuth.CheckAdmin(w, r)
	}

	user := GetUserFromContext(r.Context())
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Not authenticated. Please log in again."}`))
		return false
	}
	if !user.IsAdmin() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"Admin access required"}`))
		return false
	}
	return true
}
