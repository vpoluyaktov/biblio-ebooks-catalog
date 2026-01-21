package server

import (
	"net/http"
	"os"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	html, err := os.ReadFile("web/templates/index.html")
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html)
}

func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	s.handleIndex(w, r)
}
