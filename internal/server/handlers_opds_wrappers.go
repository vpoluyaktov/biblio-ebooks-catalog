package server

import (
	"net/http"
)

// Wrapper functions for OPDS handlers that accept IDs as parameters

func (s *Server) handleOPDSAuthorsByLetterWithParams(w http.ResponseWriter, r *http.Request, libID int64, letter string) {
	// The original handler uses chi.URLParam, so we need to call it directly with the extracted params
	// For now, just call the original handler - it will extract from context
	s.handleOPDSAuthorsByLetter(w, r)
}

func (s *Server) handleOPDSAuthorWithParams(w http.ResponseWriter, r *http.Request, libID int64, authorID int64) {
	// Call the original handler
	s.handleOPDSAuthor(w, r)
}

func (s *Server) handleOPDSSeriesBooksWithParams(w http.ResponseWriter, r *http.Request, libID int64, seriesID int64) {
	// Call the original handler
	s.handleOPDSSeriesBooks(w, r)
}

func (s *Server) handleOPDSGenreBooksWithParams(w http.ResponseWriter, r *http.Request, libID int64, genreIDStr string) {
	// Call the original handler
	s.handleOPDSGenreBooks(w, r)
}

func (s *Server) handleOPDSBookWithParams(w http.ResponseWriter, r *http.Request, libID int64, bookID int64, format string) {
	// Call the original handler
	s.handleOPDSBook(w, r)
}

func (s *Server) handleOPDSCoverWithParams(w http.ResponseWriter, r *http.Request, libID int64, bookID int64) {
	// Call the original handler
	s.handleOPDSCover(w, r)
}

func (s *Server) handleOPDSAnnotationWithParams(w http.ResponseWriter, r *http.Request, libID int64, bookID int64) {
	// Call the original handler
	s.handleOPDSAnnotation(w, r)
}
