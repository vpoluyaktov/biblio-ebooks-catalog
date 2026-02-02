package auth

import (
	"biblio-catalog/internal/db"
)

// Manager handles authentication using Biblio Auth service
type Manager struct {
	internalAuth *Auth              // For Basic Auth (OPDS)
	biblioAuth   *BiblioAuthClient  // For web UI authentication
}

// NewManager creates a new authentication manager
func NewManager(database *db.DB, biblioAuthURL string) (*Manager, error) {
	return &Manager{
		internalAuth: New(database),
		biblioAuth:   NewBiblioAuthClient(biblioAuthURL),
	}, nil
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
	return m.biblioAuth.ValidateSession(token)
}

// GetLoginURL returns the Biblio Auth login URL
func (m *Manager) GetLoginURL(returnURL string) string {
	return m.biblioAuth.GetLoginURL(returnURL)
}
