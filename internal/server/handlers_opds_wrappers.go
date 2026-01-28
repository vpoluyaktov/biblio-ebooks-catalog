package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Wrapper functions for OPDS handlers that accept IDs as parameters
// These set chi context so the original handlers can use chi.URLParam()

func (s *Server) handleOPDSAuthorsByLetterWithParams(w http.ResponseWriter, r *http.Request, libID int64, letter string) {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("libID", strconv.FormatInt(libID, 10))
	rctx.URLParams.Add("letter", letter)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	s.handleOPDSAuthorsByLetter(w, r)
}

func (s *Server) handleOPDSAuthorWithParams(w http.ResponseWriter, r *http.Request, libID int64, authorID int64) {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("libID", strconv.FormatInt(libID, 10))
	rctx.URLParams.Add("authorID", strconv.FormatInt(authorID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	s.handleOPDSAuthor(w, r)
}

func (s *Server) handleOPDSSeriesBooksWithParams(w http.ResponseWriter, r *http.Request, libID int64, seriesID int64) {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("libID", strconv.FormatInt(libID, 10))
	rctx.URLParams.Add("seriesID", strconv.FormatInt(seriesID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	s.handleOPDSSeriesBooks(w, r)
}

func (s *Server) handleOPDSGenreBooksWithParams(w http.ResponseWriter, r *http.Request, libID int64, genreIDStr string) {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("libID", strconv.FormatInt(libID, 10))
	rctx.URLParams.Add("genreID", genreIDStr)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	s.handleOPDSGenreBooks(w, r)
}

func (s *Server) handleOPDSBookWithParams(w http.ResponseWriter, r *http.Request, libID int64, bookID int64, format string) {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("libID", strconv.FormatInt(libID, 10))
	rctx.URLParams.Add("bookID", strconv.FormatInt(bookID, 10))
	rctx.URLParams.Add("format", format)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	s.handleOPDSBook(w, r)
}

func (s *Server) handleOPDSCoverWithParams(w http.ResponseWriter, r *http.Request, libID int64, bookID int64) {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("libID", strconv.FormatInt(libID, 10))
	rctx.URLParams.Add("bookID", strconv.FormatInt(bookID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	s.handleOPDSCover(w, r)
}

func (s *Server) handleOPDSAnnotationWithParams(w http.ResponseWriter, r *http.Request, libID int64, bookID int64) {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("libID", strconv.FormatInt(libID, 10))
	rctx.URLParams.Add("bookID", strconv.FormatInt(bookID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	s.handleOPDSAnnotation(w, r)
}
