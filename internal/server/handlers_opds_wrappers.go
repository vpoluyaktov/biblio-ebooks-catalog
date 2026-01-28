package server

import (
	"net/http"
)

// Wrapper functions for OPDS handlers that accept IDs as parameters
// These call the Direct versions that don't use chi context

func (s *Server) handleOPDSAuthorsByLetterWithParams(w http.ResponseWriter, r *http.Request, libID int64, letter string) {
	s.handleOPDSAuthorsByLetterDirect(w, r, libID, letter)
}

func (s *Server) handleOPDSAuthorWithParams(w http.ResponseWriter, r *http.Request, libID int64, authorID int64) {
	s.handleOPDSAuthorDirect(w, r, libID, authorID)
}

func (s *Server) handleOPDSSeriesBooksWithParams(w http.ResponseWriter, r *http.Request, libID int64, seriesID int64) {
	s.handleOPDSSeriesBooksDirect(w, r, libID, seriesID)
}

func (s *Server) handleOPDSGenreBooksWithParams(w http.ResponseWriter, r *http.Request, libID int64, genreIDStr string) {
	s.handleOPDSGenreBooksDirect(w, r, libID, genreIDStr)
}

func (s *Server) handleOPDSBookWithParams(w http.ResponseWriter, r *http.Request, libID int64, bookID int64, format string) {
	s.handleOPDSBookDirect(w, r, libID, bookID, format)
}

func (s *Server) handleOPDSCoverWithParams(w http.ResponseWriter, r *http.Request, libID int64, bookID int64) {
	s.handleOPDSCoverDirect(w, r, libID, bookID)
}

func (s *Server) handleOPDSAnnotationWithParams(w http.ResponseWriter, r *http.Request, libID int64, bookID int64) {
	s.handleOPDSAnnotationDirect(w, r, libID, bookID)
}
