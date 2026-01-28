package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"biblio-opds-server/internal/auth"
	"biblio-opds-server/internal/config"
	"biblio-opds-server/internal/db"
)

type Server struct {
	config *config.Config
	db     *db.DB
	auth   *auth.Auth
	router *chi.Mux
}

func New(cfg *config.Config, database *db.DB) *Server {
	s := &Server{
		config: cfg,
		db:     database,
		auth:   auth.New(database),
		router: chi.NewRouter(),
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(s.corsMiddleware)
	s.router.Use(middleware.Compress(5))

	if s.config.Auth.Enabled {
		s.router.Use(s.basicAuth)
	}
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		next.ServeHTTP(w, r)
	})
}

func (s *Server) basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != s.config.Auth.User || pass != s.config.Auth.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="fb2-server"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setupRoutes() {
	basePath := s.config.Server.BasePath
	if basePath == "" {
		basePath = "/"
	}
	// Ensure base path ends with /
	if basePath != "/" && basePath[len(basePath)-1] != '/' {
		basePath += "/"
	}

	// If we have a base path, mount everything under it
	if basePath != "/" {
		s.router.Route(basePath[:len(basePath)-1], func(r chi.Router) {
			s.setupRoutesWithBase(r)
		})
		// Redirect from base path without trailing slash
		s.router.Get(basePath[:len(basePath)-1], func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, basePath, http.StatusMovedPermanently)
		})
	} else {
		s.setupRoutesWithBase(s.router)
	}
}

func (s *Server) setupRoutesWithBase(r chi.Router) {
	// Static files
	basePath := s.config.Server.BasePath
	if basePath == "" {
		basePath = "/"
	}
	if basePath != "/" && basePath[len(basePath)-1] != '/' {
		basePath += "/"
	}

	// Create file server
	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When mounted under base path with chi.Route(), the full path includes base path
		// Strip both base path and /static prefix
		path := r.URL.Path
		if basePath != "/" {
			path = strings.TrimPrefix(path, basePath[:len(basePath)-1])
		}
		path = strings.TrimPrefix(path, "/static")
		r.URL.Path = path
		fileServer.ServeHTTP(w, r)
	}))

	// Web UI routes
	r.Get("/", s.handleIndex)
	r.Get("/library/{libID}", s.handleLibrary)

	// OPDS routes (support both session auth for web UI and Basic Auth for e-readers)
	r.Route("/opds", func(r chi.Router) {
		r.Use(s.auth.SessionMiddleware)
		r.Use(s.auth.BasicAuthMiddleware)
		r.Get("/{libID}", s.handleOPDSRoot)
		r.Get("/{libID}/authors", s.handleOPDSAuthors)
		r.Get("/{libID}/authors/{letter}", s.handleOPDSAuthorsByLetter)
		r.Get("/{libID}/author/{authorID}", s.handleOPDSAuthor)
		r.Get("/{libID}/series", s.handleOPDSSeries)
		r.Get("/{libID}/series/{seriesID}", s.handleOPDSSeriesBooks)
		r.Get("/{libID}/genres", s.handleOPDSGenres)
		r.Get("/{libID}/genres/{genreID}", s.handleOPDSGenreBooks)
		r.Get("/{libID}/book/{bookID}/{format}", s.handleOPDSBook)
		r.Get("/{libID}/covers/{bookID}/cover.jpg", s.handleOPDSCover)
		r.Get("/{libID}/annotation/{bookID}", s.handleOPDSAnnotation)
		r.Get("/{libID}/search", s.handleOPDSSearch)
		r.Get("/{libID}/opensearch.xml", s.handleOpenSearch)
	})

	// REST API routes
	r.Route("/api", func(r chi.Router) {
		// Apply session middleware to all API routes
		r.Use(s.auth.SessionMiddleware)

		// Public auth endpoints
		r.Get("/setup/check", s.handleSetupCheck)
		r.Post("/setup", s.handleSetup)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleCurrentUser)

		// Library routes (read-only for all authenticated users)
		r.Get("/libraries", s.apiGetLibraries)
		r.Get("/libraries/{libID}", s.apiGetLibrary)
		r.Get("/libraries/{libID}/books", s.apiGetBooks)
		r.Get("/libraries/{libID}/authors", s.apiGetAuthors)
		r.Get("/libraries/{libID}/series", s.apiGetSeries)
		r.Get("/books/{bookID}", s.apiGetBook)
		r.Get("/authors/{authorID}", s.apiGetAuthor)
		r.Get("/genres", s.apiGetGenres)

		// Admin-only routes
		r.Group(func(r chi.Router) {
			r.Use(s.auth.RequireAdmin)

			// File browser for path selection
			r.Get("/browse", s.apiBrowseFiles)

			// Library management
			r.Post("/libraries", s.apiCreateLibrary)
			r.Get("/libraries/import", s.apiImportLibrarySSE)
			r.Post("/libraries/reindex", s.apiReindex)
			r.Put("/libraries/{libID}", s.apiUpdateLibrary)
			r.Delete("/libraries/{libID}", s.apiDeleteLibrary)
			r.Get("/libraries/{libID}/stats", s.apiGetLibraryStats)
			r.Put("/libraries/{libID}/toggle", s.apiToggleLibrary)

			// User management
			r.Get("/users", s.handleGetUsers)
			r.Post("/users", s.handleCreateUser)
			r.Get("/users/{userID}", s.handleGetUser)
			r.Put("/users/{userID}/password", s.handleUpdateUserPassword)
			r.Put("/users/{userID}/role", s.handleUpdateUserRole)
			r.Delete("/users/{userID}", s.handleDeleteUser)
		})
	})
}

func (s *Server) Run(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
