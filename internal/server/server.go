package server

import (
	"fmt"
	"net/http"

	"biblio-catalog/internal/auth"
	"biblio-catalog/internal/config"
	"biblio-catalog/internal/db"
)

type Server struct {
	config      *config.Config
	db          *db.DB
	auth        *auth.Auth
	authManager *auth.Manager
	mux         *http.ServeMux
}

func New(cfg *config.Config, database *db.DB) (*Server, error) {
	// Create auth manager with configured mode
	authManager, err := auth.NewManager(cfg.Auth.Mode, database, cfg.BiblioAuth.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth manager: %w", err)
	}

	s := &Server{
		config:      cfg,
		db:          database,
		auth:        auth.New(database), // Keep for backward compatibility
		authManager: authManager,
	}
	return s, nil
}

// apiURL generates a URL with the configured base path
func (s *Server) apiURL(path string) string {
	return s.config.Server.BasePath + path
}

func (s *Server) setupRoutes() *http.ServeMux {
	// Note: Routes are registered WITH the base path prefix.
	// Nginx preserves the full path when forwarding requests.
	// This allows the service to work on a sub-path.

	mux := http.NewServeMux()
	basePath := s.config.Server.BasePath

	// Static files
	staticPath := basePath + "/static/"
	mux.Handle(staticPath, http.StripPrefix(basePath+"/static", http.FileServer(http.Dir("web/static"))))

	// Web UI routes
	if basePath == "" {
		mux.HandleFunc("/", s.handleIndex)
		mux.HandleFunc("/library/", s.handleLibrary)
		mux.HandleFunc("/reader", s.handleReader)
	} else {
		mux.HandleFunc(basePath+"/", s.handleIndex)
		mux.HandleFunc(basePath, s.handleIndex)
		mux.HandleFunc(basePath+"/library/", s.handleLibrary)
		mux.HandleFunc(basePath+"/reader", s.handleReader)
	}

	// OPDS routes - /opds/opds/{libID}/...
	mux.HandleFunc(basePath+"/opds/", s.handleOPDSRoutes)

	// API routes
	mux.HandleFunc(basePath+"/api/", s.handleAPIRoutes)

	s.mux = mux
	return mux
}

// handleOPDSRoutes routes OPDS requests
func (s *Server) handleOPDSRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	basePath := s.config.Server.BasePath

	// Strip base path and /opds prefix
	opdsPrefix := basePath + "/opds"
	if len(path) <= len(opdsPrefix) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	path = path[len(opdsPrefix):]

	// Check auth (session or basic auth for e-readers)
	if !s.authManager.CheckSessionOrBasicAuth(w, r) {
		return
	}

	// Route based on path pattern
	// Pattern: /{libID}/...
	if len(path) > 1 && path[0] == '/' {
		remaining := path[1:]
		s.handleOPDSLibraryRoutes(w, r, remaining)
		return
	}

	http.NotFound(w, r)
}

// handleOPDSLibraryRoutes handles /opds/{libID}/... routes
func (s *Server) handleOPDSLibraryRoutes(w http.ResponseWriter, r *http.Request, remaining string) {
	// Extract library ID
	var idStr string
	var action string
	if idx := indexOf(remaining, "/"); idx != -1 {
		idStr = remaining[:idx]
		action = remaining[idx+1:]
	} else {
		idStr = remaining
	}

	// Parse library ID
	libID, err := parseInt64(idStr)
	if err != nil {
		http.Error(w, "Invalid library ID", http.StatusBadRequest)
		return
	}

	// Route based on action
	if action == "" {
		s.handleOPDSRoot(w, r)
	} else if action == "authors" {
		s.handleOPDSAuthors(w, r)
	} else if indexOf(action, "authors/") == 0 {
		// /authors/{letter}
		letter := action[8:]
		s.handleOPDSAuthorsByLetterWithParams(w, r, libID, letter)
	} else if indexOf(action, "author/") == 0 {
		// /author/{authorID}
		authorIDStr := action[7:]
		if authorID, err := parseInt64(authorIDStr); err == nil {
			s.handleOPDSAuthorWithParams(w, r, libID, authorID)
		} else {
			http.Error(w, "Invalid author ID", http.StatusBadRequest)
		}
	} else if action == "series" {
		s.handleOPDSSeries(w, r)
	} else if indexOf(action, "series/") == 0 {
		// /series/{seriesID}
		seriesIDStr := action[7:]
		if seriesID, err := parseInt64(seriesIDStr); err == nil {
			s.handleOPDSSeriesBooksWithParams(w, r, libID, seriesID)
		} else {
			http.Error(w, "Invalid series ID", http.StatusBadRequest)
		}
	} else if action == "genres" {
		s.handleOPDSGenres(w, r)
	} else if indexOf(action, "genres/") == 0 {
		// /genres/{genreID}
		genreIDStr := action[7:]
		s.handleOPDSGenreBooksWithParams(w, r, libID, genreIDStr)
	} else if indexOf(action, "book/") == 0 {
		// /book/{bookID}/{format}
		parts := action[5:]
		if idx := indexOf(parts, "/"); idx != -1 {
			bookIDStr := parts[:idx]
			format := parts[idx+1:]
			if bookID, err := parseInt64(bookIDStr); err == nil {
				s.handleOPDSBookWithParams(w, r, libID, bookID, format)
			} else {
				http.Error(w, "Invalid book ID", http.StatusBadRequest)
			}
		} else {
			http.Error(w, "Format required", http.StatusBadRequest)
		}
	} else if indexOf(action, "covers/") == 0 {
		// /covers/{bookID}/cover.jpg
		parts := action[7:]
		if idx := indexOf(parts, "/"); idx != -1 {
			bookIDStr := parts[:idx]
			if bookID, err := parseInt64(bookIDStr); err == nil {
				s.handleOPDSCoverWithParams(w, r, libID, bookID)
			} else {
				http.Error(w, "Invalid book ID", http.StatusBadRequest)
			}
		} else {
			http.Error(w, "Invalid cover path", http.StatusBadRequest)
		}
	} else if indexOf(action, "annotation/") == 0 {
		// /annotation/{bookID}
		bookIDStr := action[11:]
		if bookID, err := parseInt64(bookIDStr); err == nil {
			s.handleOPDSAnnotationWithParams(w, r, libID, bookID)
		} else {
			http.Error(w, "Invalid book ID", http.StatusBadRequest)
		}
	} else if action == "search" {
		s.handleOPDSSearch(w, r)
	} else if action == "opensearch.xml" {
		s.handleOpenSearch(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// handleAPIRoutes routes API requests
func (s *Server) handleAPIRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	basePath := s.config.Server.BasePath

	// Strip base path and /api prefix
	apiPrefix := basePath + "/api"
	if len(path) <= len(apiPrefix) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	path = path[len(apiPrefix):]

	// Public endpoints (no auth required)
	if path == "/setup/check" && r.Method == "GET" {
		s.handleSetupCheck(w, r)
		return
	}
	if path == "/setup" && r.Method == "POST" {
		s.handleSetup(w, r)
		return
	}
	// Auth info endpoint (public - returns auth mode and status)
	if path == "/auth/info" && r.Method == "GET" {
		s.handleAuthInfo(w, r)
		return
	}
	// Internal auth endpoints
	if path == "/auth/login" && r.Method == "POST" {
		s.handleLogin(w, r)
		return
	}

	// All other routes require session auth
	if !s.authManager.CheckSession(w, r) {
		return
	}

	// Auth endpoints
	if path == "/auth/logout" && r.Method == "POST" {
		s.handleLogout(w, r)
		return
	}
	if path == "/auth/me" && r.Method == "GET" {
		s.handleCurrentUser(w, r)
		return
	}

	// Library endpoints
	if path == "/libraries" && r.Method == "GET" {
		s.apiGetLibraries(w, r)
		return
	}
	if path == "/genres" && r.Method == "GET" {
		s.apiGetGenres(w, r)
		return
	}

	// Special library routes (must be before /libraries/ pattern matching)
	if path == "/libraries/import" && r.Method == "GET" {
		// Admin only
		if !s.authManager.CheckAdmin(w, r) {
			return
		}
		s.apiImportLibrarySSE(w, r)
		return
	}
	if path == "/libraries/reindex" && r.Method == "POST" {
		// Admin only
		if !s.authManager.CheckAdmin(w, r) {
			return
		}
		s.apiReindex(w, r)
		return
	}

	// Routes with IDs - delegate to specific handlers
	if len(path) > len("/libraries/") && path[:len("/libraries/")] == "/libraries/" {
		s.handleLibraryRoutes(w, r, path[len("/libraries/"):])
		return
	}
	if len(path) > len("/books/") && path[:len("/books/")] == "/books/" {
		s.handleBookRoutes(w, r, path[len("/books/"):])
		return
	}
	if len(path) > len("/authors/") && path[:len("/authors/")] == "/authors/" {
		s.handleAuthorRoutes(w, r, path[len("/authors/"):])
		return
	}
	if len(path) > len("/users/") && path[:len("/users/")] == "/users/" {
		// Admin only
		if !s.authManager.CheckAdmin(w, r) {
			return
		}
		s.handleUserRoutes(w, r, path[len("/users/"):])
		return
	}

	// Admin-only routes
	if !s.authManager.CheckAdmin(w, r) {
		return
	}

	if path == "/browse" && r.Method == "GET" {
		s.apiBrowseFiles(w, r)
		return
	}
	if path == "/users" && r.Method == "GET" {
		s.handleGetUsers(w, r)
		return
	}
	if path == "/users" && r.Method == "POST" {
		s.handleCreateUser(w, r)
		return
	}

	http.NotFound(w, r)
}

// handleLibraryRoutes handles /api/libraries/{id}/... routes
func (s *Server) handleLibraryRoutes(w http.ResponseWriter, r *http.Request, remaining string) {
	// Extract library ID
	var idStr string
	var action string
	if idx := indexOf(remaining, "/"); idx != -1 {
		idStr = remaining[:idx]
		action = remaining[idx+1:]
	} else {
		idStr = remaining
	}

	// Parse library ID
	libID, err := parseInt64(idStr)
	if err != nil {
		s.jsonError(w, "Invalid library ID", http.StatusBadRequest)
		return
	}

	// Route based on action
	if action == "" && r.Method == "GET" {
		s.apiGetLibraryWithID(w, r, libID)
	} else if action == "books" && r.Method == "GET" {
		s.apiGetBooksWithID(w, r, libID)
	} else if action == "authors" && r.Method == "GET" {
		s.apiGetAuthorsWithID(w, r, libID)
	} else if action == "series" && r.Method == "GET" {
		s.apiGetSeriesWithID(w, r, libID)
	} else if action == "stats" && r.Method == "GET" {
		if !s.authManager.CheckAdmin(w, r) {
			return
		}
		s.apiGetLibraryStatsWithID(w, r, libID)
	} else if action == "toggle" && r.Method == "PUT" {
		if !s.authManager.CheckAdmin(w, r) {
			return
		}
		s.apiToggleLibraryWithID(w, r, libID)
	} else if action == "" && r.Method == "PUT" {
		if !s.authManager.CheckAdmin(w, r) {
			return
		}
		s.apiUpdateLibraryWithID(w, r, libID)
	} else if action == "" && r.Method == "DELETE" {
		if !s.authManager.CheckAdmin(w, r) {
			return
		}
		s.apiDeleteLibraryWithID(w, r, libID)
	} else {
		http.NotFound(w, r)
	}
}

// handleBookRoutes handles /api/books/{id}/... routes
func (s *Server) handleBookRoutes(w http.ResponseWriter, r *http.Request, remaining string) {
	var idStr string
	var action string
	if idx := indexOf(remaining, "/"); idx != -1 {
		idStr = remaining[:idx]
		action = remaining[idx+1:]
	} else {
		idStr = remaining
	}

	bookID, err := parseInt64(idStr)
	if err != nil {
		s.jsonError(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	if action == "content" && r.Method == "GET" {
		s.apiGetBookContentWithID(w, r, bookID)
	} else if action == "" && r.Method == "GET" {
		s.apiGetBookWithID(w, r, bookID)
	} else {
		http.NotFound(w, r)
	}
}

// handleAuthorRoutes handles /api/authors/{id}/... routes
func (s *Server) handleAuthorRoutes(w http.ResponseWriter, r *http.Request, remaining string) {
	authorID, err := parseInt64(remaining)
	if err != nil {
		s.jsonError(w, "Invalid author ID", http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		s.apiGetAuthorWithID(w, r, authorID)
	} else {
		http.NotFound(w, r)
	}
}

// handleUserRoutes handles /api/users/{id}/... routes
func (s *Server) handleUserRoutes(w http.ResponseWriter, r *http.Request, remaining string) {
	var idStr string
	var action string
	if idx := indexOf(remaining, "/"); idx != -1 {
		idStr = remaining[:idx]
		action = remaining[idx+1:]
	} else {
		idStr = remaining
	}

	userID, err := parseInt64(idStr)
	if err != nil {
		s.jsonError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if action == "" && r.Method == "GET" {
		s.handleGetUserWithID(w, r, userID)
	} else if action == "password" && r.Method == "PUT" {
		s.handleUpdateUserPasswordWithID(w, r, userID)
	} else if action == "role" && r.Method == "PUT" {
		s.handleUpdateUserRoleWithID(w, r, userID)
	} else if action == "" && r.Method == "DELETE" {
		s.handleDeleteUserWithID(w, r, userID)
	} else {
		http.NotFound(w, r)
	}
}

// Helper functions
func indexOf(s string, substr string) int {
	for i := 0; i < len(s); i++ {
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func parseInt64(s string) (int64, error) {
	var result int64
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("invalid number")
		}
		result = result*10 + int64(s[i]-'0')
	}
	return result, nil
}

func (s *Server) Run(addr string) error {
	mux := s.setupRoutes()
	return http.ListenAndServe(addr, mux)
}
