package server

import (
	"crypto/md5"
	"encoding/hex"
	"html/template"
	"net/http"
	"os"
	"sync"
)

// staticVersion holds the computed hash of static files for cache busting
var (
	staticVersion     string
	staticVersionOnce sync.Once
)

// computeStaticVersion computes an MD5 hash of static JS and CSS files
// This is called once at first request and cached for the server lifetime
func computeStaticVersion() string {
	staticVersionOnce.Do(func() {
		h := md5.New()

		// Hash all JS files
		if jsContent, err := os.ReadFile("web/static/js/app.js"); err == nil {
			h.Write(jsContent)
		}
		if mobileJsContent, err := os.ReadFile("web/static/js/mobile.js"); err == nil {
			h.Write(mobileJsContent)
		}

		// Hash all CSS files
		if cssContent, err := os.ReadFile("web/static/css/style.css"); err == nil {
			h.Write(cssContent)
		}
		if mobileCssContent, err := os.ReadFile("web/static/css/mobile.css"); err == nil {
			h.Write(mobileCssContent)
		}

		// Use first 8 characters of hash for brevity
		staticVersion = hex.EncodeToString(h.Sum(nil))[:8]
	})
	return staticVersion
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	data := struct {
		Version string
	}{
		Version: computeStaticVersion(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	s.handleIndex(w, r)
}
