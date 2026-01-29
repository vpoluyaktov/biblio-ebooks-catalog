package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"biblio-catalog/internal/db"
	"biblio-catalog/internal/importer"
)

// API handlers for opds-server

func (s *Server) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Server) apiGetLibraries(w http.ResponseWriter, r *http.Request) {
	libraries, err := s.db.GetLibraries()
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if libraries == nil {
		libraries = []db.Library{}
	}
	s.jsonResponse(w, libraries)
}

type ImportLibraryRequest struct {
	Name            string `json:"name"`
	InpxPath        string `json:"inpx_path"`
	LibraryPath     string `json:"library_path"`
	FirstAuthorOnly bool   `json:"first_author_only"`
}

type ImportLibraryResponse struct {
	Success   bool   `json:"success"`
	LibraryID int64  `json:"library_id,omitempty"`
	Message   string `json:"message,omitempty"`
	BookCount int    `json:"book_count,omitempty"`
}

func (s *Server) apiCreateLibrary(w http.ResponseWriter, r *http.Request) {
	var req ImportLibraryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.InpxPath == "" || req.LibraryPath == "" {
		s.jsonError(w, "Name, inpx_path, and library_path are required", http.StatusBadRequest)
		return
	}

	// Validate paths exist on server
	if _, err := os.Stat(req.InpxPath); os.IsNotExist(err) {
		s.jsonError(w, "INPX file not found: "+req.InpxPath, http.StatusBadRequest)
		return
	}
	if _, err := os.Stat(req.LibraryPath); os.IsNotExist(err) {
		s.jsonError(w, "Library path not found: "+req.LibraryPath, http.StatusBadRequest)
		return
	}

	// Import the library
	imp := importer.New(s.db)
	newLibID, err := imp.ImportINPX(req.InpxPath, req.Name, req.LibraryPath, req.FirstAuthorOnly)
	if err != nil {
		s.jsonError(w, "Import failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get book count
	bookCount, _, _, _ := s.db.GetLibraryStats(newLibID)

	s.jsonResponse(w, ImportLibraryResponse{
		Success:   true,
		LibraryID: newLibID,
		Message:   "Library imported successfully",
		BookCount: int(bookCount),
	})
}

type ImportProgress struct {
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Message string `json:"message"`
	Done    bool   `json:"done"`
	Error   string `json:"error,omitempty"`
	LibID   int64  `json:"library_id,omitempty"`
	// ZIP file progress (for dual progress bar)
	ZipCurrent  int    `json:"zip_current,omitempty"`
	ZipTotal    int    `json:"zip_total,omitempty"`
	ZipFileName string `json:"zip_filename,omitempty"`
}

func (s *Server) apiImportLibrarySSE(w http.ResponseWriter, r *http.Request) {
	// Parse request from query params for SSE
	name := r.URL.Query().Get("name")
	inpxPath := r.URL.Query().Get("inpx_path")
	libraryPath := r.URL.Query().Get("library_path")
	firstAuthorOnly := r.URL.Query().Get("first_author_only") == "true"

	// Only name and library_path are required
	if name == "" || libraryPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"name and library_path are required"}`))
		return
	}

	// Validate library path exists
	if _, err := os.Stat(libraryPath); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Library path not found"}`))
		return
	}

	// If INPX path is provided, validate it exists
	if inpxPath != "" {
		if _, err := os.Stat(inpxPath); os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"INPX file not found"}`))
			return
		}
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	sendProgress := func(p ImportProgress) {
		data, _ := json.Marshal(p)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Create context that cancels when client disconnects
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Import with progress callback
	imp := importer.New(s.db)
	imp.SetProgressCallback(func(current, total int, message string) {
		sendProgress(ImportProgress{
			Current: current,
			Total:   total,
			Message: message,
			Done:    false,
		})
	})

	var libID int64
	var err error

	// Choose import method based on whether INPX path is provided
	if inpxPath != "" {
		// INPX import
		libID, err = imp.ImportINPX(inpxPath, name, libraryPath, firstAuthorOnly)
	} else {
		// Scan import (for EPUB/FB2 files) - Streaming Import Flow
		// Create library BEFORE scanning so it exists even if scan is interrupted
		libID, err = imp.CreateLibraryForImport(name, libraryPath, firstAuthorOnly)
		if err != nil {
			sendProgress(ImportProgress{
				Done:  true,
				Error: "Failed to create library: " + err.Error(),
			})
			return
		}

		// Phase 1: File Discovery (fast - just list files)
		discovery := importer.NewFileDiscovery(libraryPath)
		discovery.SetProgressCallback(func(current, total int, message string) {
			sendProgress(ImportProgress{
				Current: current,
				Total:   total,
				Message: message,
				Done:    false,
			})
		})

		files, discErr := discovery.DiscoverFiles()
		if discErr != nil {
			sendProgress(ImportProgress{
				Done:  true,
				Error: "File discovery failed: " + discErr.Error(),
			})
			return
		}

		if len(files) == 0 {
			sendProgress(ImportProgress{
				Done:  true,
				Error: "No book files found in directory",
			})
			return
		}

		// Phase 2: Streaming Import (slow - parse and import one-by-one)
		streamingImp := importer.NewStreamingImporter(s.db, libID, libraryPath, firstAuthorOnly)
		streamingImp.SetProgressCallback(func(current, total int, message string) {
			sendProgress(ImportProgress{
				Current: current,
				Total:   total,
				Message: message,
				Done:    false,
			})
		})
		// Set ZIP progress callback for dual progress bars
		streamingImp.SetZipProgressCallback(func(fileIndex, fileTotal, zipCurrent, zipTotal int, zipFileName, message string) {
			sendProgress(ImportProgress{
				Current:     fileIndex,
				Total:       fileTotal,
				Message:     message,
				Done:        false,
				ZipCurrent:  zipCurrent,
				ZipTotal:    zipTotal,
				ZipFileName: zipFileName,
			})
		})

		// Load genre codes
		if err := streamingImp.LoadGenreCodes(); err != nil {
			sendProgress(ImportProgress{
				Done:  true,
				Error: "Failed to load genre codes: " + err.Error(),
			})
			return
		}

		// Import files with streaming
		err = streamingImp.ImportFiles(ctx, files)
		if err != nil {
			if err == context.Canceled {
				sendProgress(ImportProgress{
					Done:  true,
					Error: "Import canceled. Check library for partial results.",
				})
				log.Printf("Import canceled for library: %s", name)
				return
			}
			sendProgress(ImportProgress{
				Done:  true,
				Error: "Import failed: " + err.Error(),
			})
			return
		}

		err = nil
	}

	if err != nil {
		sendProgress(ImportProgress{
			Done:  true,
			Error: err.Error(),
		})
		return
	}

	bookCount, _, _, _ := s.db.GetLibraryStats(libID)

	sendProgress(ImportProgress{
		Current: 100,
		Total:   100,
		Message: fmt.Sprintf("Import complete: %d books", bookCount),
		Done:    true,
		LibID:   libID,
	})
}

func (s *Server) apiGetLibrary(w http.ResponseWriter, r *http.Request, id int64) {
	lib, err := s.db.GetLibrary(id)
	if err != nil {
		s.jsonError(w, "Library not found", http.StatusNotFound)
		return
	}
	s.jsonResponse(w, lib)
}

func (s *Server) apiDeleteLibrary(w http.ResponseWriter, r *http.Request, id int64) {
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
		"success": true,
		"message": "Library deleted",
		"library": lib,
	})
}

func (s *Server) apiGetBooks(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) apiGetAuthors(w http.ResponseWriter, r *http.Request, libID int64) {

	// Get pagination and filter params
	limit := 50 // Default limit for virtual scrolling
	offset := 0
	filter := r.URL.Query().Get("filter")

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	result, err := s.db.GetAuthorsFiltered(libID, filter, limit, offset)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, result)
}

func (s *Server) apiGetSeries(w http.ResponseWriter, r *http.Request, libID int64) {

	// Get pagination and filter params
	limit := 50 // Default limit for virtual scrolling
	offset := 0
	filter := r.URL.Query().Get("filter")

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	result, err := s.db.GetSeriesFiltered(libID, filter, limit, offset)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, result)
}

func (s *Server) apiGetBook(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) apiGetAuthor(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) apiGetGenres(w http.ResponseWriter, r *http.Request) {
	genres, err := s.db.GetGenres()
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, genres)
}

type FileInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}

type BrowseResponse struct {
	Path    string     `json:"path"`
	Parent  string     `json:"parent,omitempty"`
	Entries []FileInfo `json:"entries"`
}

func (s *Server) apiBrowseFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	fileType := r.URL.Query().Get("type") // "dir", "file", or "inpx"

	if path == "" {
		path = "/"
	}

	// Clean and validate path
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		s.jsonError(w, "Path not found: "+path, http.StatusBadRequest)
		return
	}

	if !info.IsDir() {
		s.jsonError(w, "Path is not a directory", http.StatusBadRequest)
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		s.jsonError(w, "Cannot read directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(path, entry.Name())

		// Filter based on type
		if fileType == "dir" && !entry.IsDir() {
			continue
		}
		if fileType == "inpx" && !entry.IsDir() && !strings.HasSuffix(strings.ToLower(entry.Name()), ".inpx") {
			continue
		}
		if fileType == "file" && entry.IsDir() {
			// Still show directories for navigation
		}

		files = append(files, FileInfo{
			Name:  entry.Name(),
			Path:  fullPath,
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		})
	}

	// Sort: directories first, then alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	parent := ""
	if path != "/" {
		parent = filepath.Dir(path)
	}

	s.jsonResponse(w, BrowseResponse{
		Path:    path,
		Parent:  parent,
		Entries: files,
	})
}

type LibraryStatsResponse struct {
	Library     interface{} `json:"library"`
	BookCount   int64       `json:"book_count"`
	AuthorCount int64       `json:"author_count"`
	SeriesCount int64       `json:"series_count"`
}

func (s *Server) apiGetLibraryStats(w http.ResponseWriter, r *http.Request, id int64) {
	lib, err := s.db.GetLibrary(id)
	if err != nil {
		s.jsonError(w, "Library not found", http.StatusNotFound)
		return
	}

	bookCount, authorCount, seriesCount, err := s.db.GetLibraryStats(id)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, LibraryStatsResponse{
		Library:     lib,
		BookCount:   bookCount,
		AuthorCount: authorCount,
		SeriesCount: seriesCount,
	})
}

func (s *Server) apiUpdateLibrary(w http.ResponseWriter, r *http.Request, id int64) {
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

func (s *Server) apiToggleLibrary(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.db.SetLibraryEnabled(id, req.Enabled); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lib, _ := s.db.GetLibrary(id)
	s.jsonResponse(w, lib)
}

// ReindexRequest represents a request to export a library to INPX
type ReindexRequest struct {
	LibraryID   int64  `json:"library_id,omitempty"`
	LibraryName string `json:"library_name,omitempty"`
	OutputPath  string `json:"output_path"`
}

// ReindexResponse represents the response from a reindex operation
type ReindexResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
	OutputPath string `json:"output_path,omitempty"`
	BookCount  int    `json:"book_count,omitempty"`
}

func (s *Server) apiReindex(w http.ResponseWriter, r *http.Request) {
	var req ReindexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.LibraryID == 0 && req.LibraryName == "" {
		s.jsonError(w, "Either library_id or library_name is required", http.StatusBadRequest)
		return
	}

	if req.OutputPath == "" {
		s.jsonError(w, "Output path is required", http.StatusBadRequest)
		return
	}

	// Validate output directory exists
	outputDir := filepath.Dir(req.OutputPath)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		s.jsonError(w, "Output directory not found: "+outputDir, http.StatusBadRequest)
		return
	}

	writer := importer.NewINPXWriter(s.db)

	var err error
	var libraryID int64

	if req.LibraryID > 0 {
		libraryID = req.LibraryID
		err = writer.ExportLibraryToINPX(req.LibraryID, req.OutputPath)
	} else {
		// Find library by name
		libraries, libErr := s.db.GetLibraries()
		if libErr != nil {
			s.jsonError(w, "Failed to get libraries: "+libErr.Error(), http.StatusInternalServerError)
			return
		}

		for _, lib := range libraries {
			if lib.Name == req.LibraryName {
				libraryID = lib.ID
				break
			}
		}

		if libraryID == 0 {
			s.jsonError(w, "Library not found: "+req.LibraryName, http.StatusNotFound)
			return
		}

		err = writer.ExportLibraryToINPX(libraryID, req.OutputPath)
	}

	if err != nil {
		s.jsonError(w, "Reindex failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get book count
	bookCount, _, _, _ := s.db.GetLibraryStats(libraryID)

	s.jsonResponse(w, ReindexResponse{
		Success:    true,
		Message:    fmt.Sprintf("Successfully exported %d books to INPX", bookCount),
		OutputPath: req.OutputPath,
		BookCount:  int(bookCount),
	})
}
