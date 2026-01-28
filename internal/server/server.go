package server

import (
	"log"
	"net/http"
	"strings"

	"biblio-opds-server/internal/auth"
	"biblio-opds-server/internal/config"
	"biblio-opds-server/internal/db"
)

type Server struct {
	config *config.Config
	db     *db.DB
	auth   *auth.Auth
	mux    *http.ServeMux
}

func New(cfg *config.Config, database *db.DB) *Server {
	s := &Server{
		config: cfg,
		db:     database,
		auth:   auth.New(database),
		mux:    http.NewServeMux(),
	}

	s.setupRoutes()

	return s
}

func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next(w, r)
	}
}

func (s *Server) setupRoutes() {
	// Note: Routes are registered WITH the base path prefix.
	// Nginx preserves the full path when forwarding requests.
	// This allows the service to work on a sub-path.

	basePath := s.config.Server.BasePath
	if basePath == "" {
		basePath = ""
	}

	// Static files - serve from web/static directory
	staticPath := basePath + "/static/"
	s.mux.Handle(staticPath, http.StripPrefix(basePath+"/static", http.FileServer(http.Dir("web/static"))))

	// Web UI routes
	s.mux.HandleFunc(basePath+"/", s.loggingMiddleware(s.corsMiddleware(s.handleIndex)))
	s.mux.HandleFunc(basePath+"/library/", s.loggingMiddleware(s.corsMiddleware(s.handleLibrary)))

	// OPDS routes (support both session auth for web UI and Basic Auth for e-readers)
	s.mux.HandleFunc(basePath+"/opds/", s.loggingMiddleware(s.corsMiddleware(s.handleOPDSRoutes)))

	// REST API routes
	s.mux.HandleFunc(basePath+"/api/", s.loggingMiddleware(s.corsMiddleware(s.handleAPIRoutes)))
}

// handleOPDSRoutes routes OPDS-specific requests
func (s *Server) handleOPDSRoutes(w http.ResponseWriter, r *http.Request) {
	// Apply auth middleware
	if !s.auth.CheckSessionOrBasicAuth(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, s.config.Server.BasePath+"/opds")

	// Route based on path patterns
	if strings.HasSuffix(path, "/authors") {
		s.handleOPDSAuthors(w, r)
	} else if strings.Contains(path, "/authors/") {
		s.handleOPDSAuthorsByLetter(w, r)
	} else if strings.Contains(path, "/author/") {
		s.handleOPDSAuthor(w, r)
	} else if strings.HasSuffix(path, "/series") {
		s.handleOPDSSeries(w, r)
	} else if strings.Contains(path, "/series/") {
		s.handleOPDSSeriesBooks(w, r)
	} else if strings.HasSuffix(path, "/genres") {
		s.handleOPDSGenres(w, r)
	} else if strings.Contains(path, "/genres/") {
		s.handleOPDSGenreBooks(w, r)
	} else if strings.Contains(path, "/book/") {
		s.handleOPDSBook(w, r)
	} else if strings.Contains(path, "/covers/") {
		s.handleOPDSCover(w, r)
	} else if strings.Contains(path, "/annotation/") {
		s.handleOPDSAnnotation(w, r)
	} else if strings.HasSuffix(path, "/search") {
		s.handleOPDSSearch(w, r)
	} else if strings.HasSuffix(path, "/opensearch.xml") {
		s.handleOpenSearch(w, r)
	} else {
		s.handleOPDSRoot(w, r)
	}
}

// handleAPIRoutes routes API requests
func (s *Server) handleAPIRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, s.config.Server.BasePath+"/api")

	// Public auth endpoints (no session required)
	if path == "/setup/check" && r.Method == "GET" {
		s.handleSetupCheck(w, r)
		return
	}
	if path == "/setup" && r.Method == "POST" {
		s.handleSetup(w, r)
		return
	}
	if path == "/auth/login" && r.Method == "POST" {
		s.handleLogin(w, r)
		return
	}

	// All other API routes require session auth
	if !s.auth.CheckSession(w, r) {
		return
	}

	// Route based on path and method
	switch {
	case path == "/auth/logout" && r.Method == "POST":
		s.handleLogout(w, r)
	case path == "/auth/me" && r.Method == "GET":
		s.handleCurrentUser(w, r)
	case path == "/libraries" && r.Method == "GET":
		s.apiGetLibraries(w, r)
	case strings.HasPrefix(path, "/libraries/") && strings.HasSuffix(path, "/books") && r.Method == "GET":
		s.apiGetBooks(w, r)
	case strings.HasPrefix(path, "/libraries/") && strings.HasSuffix(path, "/authors") && r.Method == "GET":
		s.apiGetAuthors(w, r)
	case strings.HasPrefix(path, "/libraries/") && strings.HasSuffix(path, "/series") && r.Method == "GET":
		s.apiGetSeries(w, r)
	case strings.HasPrefix(path, "/libraries/") && !strings.Contains(path[11:], "/") && r.Method == "GET":
		s.apiGetLibrary(w, r)
	case strings.HasPrefix(path, "/books/") && r.Method == "GET":
		s.apiGetBook(w, r)
	case strings.HasPrefix(path, "/authors/") && r.Method == "GET":
		s.apiGetAuthor(w, r)
	case path == "/genres" && r.Method == "GET":
		s.apiGetGenres(w, r)
	default:
		// Admin-only routes
		if !s.auth.CheckAdmin(w, r) {
			return
		}
		s.handleAdminAPIRoutes(w, r, path)
	}
}

// handleAdminAPIRoutes handles admin-only API routes
func (s *Server) handleAdminAPIRoutes(w http.ResponseWriter, r *http.Request, path string) {
	switch {
	case path == "/browse" && r.Method == "GET":
		s.apiBrowseFiles(w, r)
	case path == "/libraries" && r.Method == "POST":
		s.apiCreateLibrary(w, r)
	case path == "/libraries/import" && r.Method == "GET":
		s.apiImportLibrarySSE(w, r)
	case path == "/libraries/reindex" && r.Method == "POST":
		s.apiReindex(w, r)
	case strings.HasPrefix(path, "/libraries/") && r.Method == "PUT":
		if strings.HasSuffix(path, "/toggle") {
			s.apiToggleLibrary(w, r)
		} else {
			s.apiUpdateLibrary(w, r)
		}
	case strings.HasPrefix(path, "/libraries/") && r.Method == "DELETE":
		s.apiDeleteLibrary(w, r)
	case strings.HasPrefix(path, "/libraries/") && strings.HasSuffix(path, "/stats") && r.Method == "GET":
		s.apiGetLibraryStats(w, r)
	case path == "/users" && r.Method == "GET":
		s.handleGetUsers(w, r)
	case path == "/users" && r.Method == "POST":
		s.handleCreateUser(w, r)
	case strings.HasPrefix(path, "/users/") && r.Method == "GET":
		s.handleGetUser(w, r)
	case strings.HasPrefix(path, "/users/") && strings.HasSuffix(path, "/password") && r.Method == "PUT":
		s.handleUpdateUserPassword(w, r)
	case strings.HasPrefix(path, "/users/") && strings.HasSuffix(path, "/role") && r.Method == "PUT":
		s.handleUpdateUserRole(w, r)
	case strings.HasPrefix(path, "/users/") && r.Method == "DELETE":
		s.handleDeleteUser(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) Run(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}
