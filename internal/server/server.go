package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"opds-server/internal/auth"
	"opds-server/internal/config"
	"opds-server/internal/db"
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
	// Static files
	s.router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Web UI routes
	s.router.Get("/", s.handleIndex)
	s.router.Get("/library/{libID}", s.handleLibrary)

	// OPDS routes (require Basic Auth for e-readers)
	s.router.Route("/opds", func(r chi.Router) {
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
	s.router.Route("/api", func(r chi.Router) {
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
