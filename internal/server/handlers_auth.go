package server

import (
	"encoding/json"
	"net/http"
	"time"

	"biblio-catalog/internal/auth"
	"biblio-catalog/internal/db"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool     `json:"success"`
	User    *db.User `json:"user,omitempty"`
	Message string   `json:"message,omitempty"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UpdatePasswordRequest struct {
	Password string `json:"password"`
}

type UpdateRoleRequest struct {
	Role string `json:"role"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Check if internal auth mode
	if !s.authManager.IsInternalMode() {
		s.jsonError(w, "Internal login not available. Please use Keycloak authentication.", http.StatusBadRequest)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := s.authManager.Authenticate(req.Username, req.Password)
	if err != nil {
		s.jsonError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	session, err := s.authManager.CreateSession(user.ID)
	if err != nil {
		s.jsonError(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	s.jsonResponse(w, LoginResponse{
		Success: true,
		User:    user,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Check if internal auth mode
	if !s.authManager.IsInternalMode() {
		s.jsonError(w, "Internal logout not available. Please use Keycloak logout.", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session")
	if err == nil {
		s.authManager.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	s.jsonResponse(w, map[string]bool{"success": true})
}

func (s *Server) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.jsonResponse(w, map[string]interface{}{"authenticated": false})
		return
	}

	s.jsonResponse(w, map[string]interface{}{
		"authenticated": true,
		"user":          user,
	})
}

func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	if !s.authManager.IsInternalMode() {
		s.jsonError(w, "User management not available in Keycloak mode. Manage users in Keycloak Admin Console.", http.StatusBadRequest)
		return
	}
	users, err := s.authManager.GetUsers()
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, users)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if !s.authManager.IsInternalMode() {
		s.jsonError(w, "User creation not available in Keycloak mode. Create users in Keycloak Admin Console.", http.StatusBadRequest)
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		s.jsonError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = db.RoleReadonly
	}

	if req.Role != db.RoleAdmin && req.Role != db.RoleReadonly {
		s.jsonError(w, "Invalid role. Must be 'admin' or 'readonly'", http.StatusBadRequest)
		return
	}

	user, err := s.authManager.CreateUser(req.Username, req.Password, req.Role)
	if err != nil {
		if err == auth.ErrUserExists {
			s.jsonError(w, "User already exists", http.StatusConflict)
			return
		}
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, user)
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request, id int64) {
	user, err := s.authManager.GetUser(id)
	if err != nil {
		s.jsonError(w, "User not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(w, user)
}

func (s *Server) handleUpdateUserPassword(w http.ResponseWriter, r *http.Request, id int64) {
	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		s.jsonError(w, "Password is required", http.StatusBadRequest)
		return
	}

	if err := s.authManager.UpdateUserPassword(id, req.Password); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]bool{"success": true})
}

func (s *Server) handleUpdateUserRole(w http.ResponseWriter, r *http.Request, id int64) {
	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Role != db.RoleAdmin && req.Role != db.RoleReadonly {
		s.jsonError(w, "Invalid role. Must be 'admin' or 'readonly'", http.StatusBadRequest)
		return
	}

	if err := s.authManager.UpdateUserRole(id, req.Role); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]bool{"success": true})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request, id int64) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser != nil && currentUser.ID == id {
		s.jsonError(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	if err := s.authManager.DeleteUser(id); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]bool{"success": true})
}

func (s *Server) handleSetupCheck(w http.ResponseWriter, r *http.Request) {
	// In Keycloak mode, setup is not required (users managed in Keycloak)
	if !s.authManager.IsInternalMode() {
		s.jsonResponse(w, map[string]bool{"setup_required": false})
		return
	}

	hasUsers, err := s.authManager.HasUsers()
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]bool{"setup_required": !hasUsers})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if !s.authManager.IsInternalMode() {
		s.jsonError(w, "Setup not available in Keycloak mode. Users are managed in Keycloak.", http.StatusBadRequest)
		return
	}

	hasUsers, err := s.authManager.HasUsers()
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if hasUsers {
		s.jsonError(w, "Setup already completed", http.StatusBadRequest)
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		s.jsonError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// First user is always admin
	user, err := s.authManager.CreateUser(req.Username, req.Password, db.RoleAdmin)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-login after setup
	session, err := s.authManager.CreateSession(user.ID)
	if err != nil {
		s.jsonError(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	s.jsonResponse(w, map[string]interface{}{
		"success": true,
		"user":    user,
	})
}
