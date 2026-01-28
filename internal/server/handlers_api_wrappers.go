package server

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Wrapper functions that accept IDs as parameters instead of extracting from chi context

func (s *Server) apiGetLibraryWithID(w http.ResponseWriter, r *http.Request, id int64) {
	lib, err := s.db.GetLibrary(id)
	if err != nil {
		s.jsonError(w, "Library not found", http.StatusNotFound)
		return
	}
	s.jsonResponse(w, lib)
}

func (s *Server) apiDeleteLibraryWithID(w http.ResponseWriter, r *http.Request, id int64) {
	lib, err := s.db.GetLibrary(id)
	if err != nil {
		s.jsonError(w, "Library not found", http.StatusNotFound)
		return
	}

	if err := s.db.DeleteLibrary(id); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]interface{}{
		"message": "Library deleted",
		"library": lib,
	})
}

func (s *Server) apiGetBooksWithID(w http.ResponseWriter, r *http.Request, libID int64) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) apiGetAuthorsWithID(w http.ResponseWriter, r *http.Request, libID int64) {
	// Get pagination and filter params
	limit := 50
	offset := 0
	filter := r.URL.Query().Get("filter")

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	result, err := s.db.GetAuthorsFiltered(libID, filter, limit, offset)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, result)
}

func (s *Server) apiGetSeriesWithID(w http.ResponseWriter, r *http.Request, libID int64) {
	// Get pagination and filter params
	limit := 50
	offset := 0
	filter := r.URL.Query().Get("filter")

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	result, err := s.db.GetSeriesFiltered(libID, filter, limit, offset)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, result)
}

func (s *Server) apiGetLibraryStatsWithID(w http.ResponseWriter, r *http.Request, id int64) {
	lib, err := s.db.GetLibrary(id)
	if err != nil {
		s.jsonError(w, "Library not found", http.StatusNotFound)
		return
	}

	bookCount, authorCount, seriesCount, _ := s.db.GetLibraryStats(id)

	s.jsonResponse(w, map[string]interface{}{
		"library":      lib,
		"book_count":   bookCount,
		"author_count": authorCount,
		"series_count": seriesCount,
	})
}

func (s *Server) apiUpdateLibraryWithID(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		Enabled *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	lib, err := s.db.GetLibrary(id)
	if err != nil {
		s.jsonError(w, "Library not found", http.StatusNotFound)
		return
	}

	if req.Name != "" {
		lib.Name = req.Name
	}
	if req.Path != "" {
		lib.Path = req.Path
	}
	if req.Enabled != nil {
		lib.Enabled = *req.Enabled
	}

	if err := s.db.UpdateLibrary(lib); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, lib)
}

func (s *Server) apiToggleLibraryWithID(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	lib, err := s.db.GetLibrary(id)
	if err != nil {
		s.jsonError(w, "Library not found", http.StatusNotFound)
		return
	}

	lib.Enabled = req.Enabled
	if err := s.db.UpdateLibrary(lib); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, lib)
}

func (s *Server) apiGetBookWithID(w http.ResponseWriter, r *http.Request, bookID int64) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) apiGetAuthorWithID(w http.ResponseWriter, r *http.Request, authorID int64) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) handleGetUserWithID(w http.ResponseWriter, r *http.Request, id int64) {
	user, err := s.auth.GetUser(id)
	if err != nil {
		s.jsonError(w, "User not found", http.StatusNotFound)
		return
	}
	s.jsonResponse(w, user)
}

func (s *Server) handleUpdateUserPasswordWithID(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		s.jsonError(w, "Password is required", http.StatusBadRequest)
		return
	}

	if err := s.auth.UpdateUserPassword(id, req.Password); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]string{"message": "Password updated"})
}

func (s *Server) handleUpdateUserRoleWithID(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Role != "admin" && req.Role != "user" {
		s.jsonError(w, "Invalid role. Must be 'admin' or 'user'", http.StatusBadRequest)
		return
	}

	if err := s.auth.UpdateUserRole(id, req.Role); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]string{"message": "Role updated"})
}

func (s *Server) handleDeleteUserWithID(w http.ResponseWriter, r *http.Request, id int64) {
	if err := s.auth.DeleteUser(id); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]string{"message": "User deleted"})
}
