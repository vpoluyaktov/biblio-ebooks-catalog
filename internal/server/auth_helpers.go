package server

import (
	"context"
	"log"
	"net/http"

	"biblio-catalog/internal/auth"
	"biblio-catalog/internal/db"
)

// checkSessionByMode validates authentication based on the configured auth mode
func (s *Server) checkSessionByMode(w http.ResponseWriter, r *http.Request) bool {
	// In biblio-auth mode, check for auth_token cookie and validate with Biblio Auth
	if s.authManager.IsBiblioAuthMode() {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			// Log for debugging
			log.Printf("No auth_token cookie found: %v (cookies: %v)", err, r.Cookies())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Not authenticated. Please log in."}`))
			return false
		}

		log.Printf("Found auth_token cookie, validating with Biblio Auth")

		// Validate token with Biblio Auth
		userInfo, err := s.authManager.ValidateBiblioAuthSession(cookie.Value)
		if err != nil {
			log.Printf("Biblio Auth validation failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Session expired. Please log in again."}`))
			return false
		}

		log.Printf("Biblio Auth validation successful: user=%s, groups=%v", userInfo.Username, userInfo.Groups)

		// Convert Biblio Auth UserInfo to internal db.User for context
		// Note: We create a temporary user object for the context
		user := &db.User{
			ID:       int64(userInfo.ID),
			Username: userInfo.Username,
			Role:     "user", // Default role
		}

		// Check if user is admin
		for _, group := range userInfo.Groups {
			if group == "admin" {
				user.Role = db.RoleAdmin
				break
			}
		}

		ctx := context.WithValue(r.Context(), auth.UserContextKey, user)
		*r = *r.WithContext(ctx)
		return true
	}

	// In internal mode, use the standard session validation
	return s.authManager.CheckSession(w, r)
}
