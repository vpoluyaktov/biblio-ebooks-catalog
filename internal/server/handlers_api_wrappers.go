package server

import (
	"net/http"
)

// Wrapper functions that accept IDs as parameters instead of extracting from chi context

func (s *Server) apiGetLibraryWithID(w http.ResponseWriter, r *http.Request, id int64) {
	s.apiGetLibrary(w, r, id)
}

func (s *Server) apiDeleteLibraryWithID(w http.ResponseWriter, r *http.Request, id int64) {
	s.apiDeleteLibrary(w, r, id)
}

func (s *Server) apiGetBooksWithID(w http.ResponseWriter, r *http.Request, libID int64) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) apiGetAuthorsWithID(w http.ResponseWriter, r *http.Request, libID int64) {
	s.apiGetAuthors(w, r, libID)
}

func (s *Server) apiGetSeriesWithID(w http.ResponseWriter, r *http.Request, libID int64) {
	s.apiGetSeries(w, r, libID)
}

func (s *Server) apiGetLibraryStatsWithID(w http.ResponseWriter, r *http.Request, id int64) {
	s.apiGetLibraryStats(w, r, id)
}

func (s *Server) apiUpdateLibraryWithID(w http.ResponseWriter, r *http.Request, id int64) {
	s.apiUpdateLibrary(w, r, id)
}

func (s *Server) apiToggleLibraryWithID(w http.ResponseWriter, r *http.Request, id int64) {
	s.apiToggleLibrary(w, r, id)
}

func (s *Server) apiGetBookWithID(w http.ResponseWriter, r *http.Request, bookID int64) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) apiGetBookContentWithID(w http.ResponseWriter, r *http.Request, bookID int64) {
	s.apiGetBookContent(w, r, bookID)
}

func (s *Server) apiGetAuthorWithID(w http.ResponseWriter, r *http.Request, authorID int64) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) handleGetUserWithID(w http.ResponseWriter, r *http.Request, id int64) {
	s.handleGetUser(w, r, id)
}

func (s *Server) handleUpdateUserPasswordWithID(w http.ResponseWriter, r *http.Request, id int64) {
	s.handleUpdateUserPassword(w, r, id)
}

func (s *Server) handleUpdateUserRoleWithID(w http.ResponseWriter, r *http.Request, id int64) {
	s.handleUpdateUserRole(w, r, id)
}

func (s *Server) handleDeleteUserWithID(w http.ResponseWriter, r *http.Request, id int64) {
	s.handleDeleteUser(w, r, id)
}
