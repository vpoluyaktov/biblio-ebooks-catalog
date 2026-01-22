package server

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"

	"opds-server/internal/bookfile"
	"opds-server/internal/db"
	"opds-server/internal/opds"
)

func (s *Server) writeOPDS(w http.ResponseWriter, feed *opds.Feed) {
	data, err := feed.ToXML()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.Write(data)
}

func (s *Server) getLibraryID(r *http.Request) int64 {
	libIDStr := chi.URLParam(r, "libID")
	libID, _ := strconv.ParseInt(libIDStr, 10, 64)
	if libID == 0 {
		libID = 1
	}
	return libID
}

func (s *Server) handleOPDSRoot(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	baseURL := fmt.Sprintf("/opds/%d", libID)

	feed := opds.NewFeed("urn:opds-server:root", "opds-server")
	feed.AddLink("self", baseURL, "application/atom+xml;profile=opds-catalog;kind=navigation")
	feed.AddLink("start", baseURL, "application/atom+xml;profile=opds-catalog;kind=navigation")
	feed.AddLink("search", baseURL+"/opensearch.xml", "application/opensearchdescription+xml")

	feed.AddNavEntry("urn:opds-server:authors", "По авторам", baseURL+"/authors")
	feed.AddNavEntry("urn:opds-server:series", "По сериям", baseURL+"/series")
	feed.AddNavEntry("urn:opds-server:genres", "По жанрам", baseURL+"/genres")

	s.writeOPDS(w, feed)
}

func (s *Server) handleOPDSAuthors(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	baseURL := fmt.Sprintf("/opds/%d", libID)

	feed := opds.NewFeed("urn:opds-server:authors", "Авторы")
	feed.AddLink("self", baseURL+"/authors", "application/atom+xml;profile=opds-catalog;kind=navigation")
	feed.AddLink("up", baseURL, "application/atom+xml;profile=opds-catalog")

	// Add alphabet navigation
	cyrillic := "АБВГДЕЖЗИЙКЛМНОПРСТУФХЦЧШЩЭЮЯ"
	latin := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for _, letter := range cyrillic {
		l := string(letter)
		feed.AddNavEntry(
			fmt.Sprintf("urn:opds-server:authors:%s", l),
			l,
			fmt.Sprintf("%s/authors/%s", baseURL, l),
		)
	}
	for _, letter := range latin {
		l := string(letter)
		feed.AddNavEntry(
			fmt.Sprintf("urn:opds-server:authors:%s", l),
			l,
			fmt.Sprintf("%s/authors/%s", baseURL, l),
		)
	}

	s.writeOPDS(w, feed)
}

func (s *Server) handleOPDSAuthorsByLetter(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	letter := chi.URLParam(r, "letter")
	// URL decode the letter for Cyrillic support
	if decoded, err := url.QueryUnescape(letter); err == nil {
		letter = decoded
	}
	baseURL := fmt.Sprintf("/opds/%d", libID)

	authors, err := s.db.GetAuthorsByLetter(libID, letter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed := opds.NewFeed(
		fmt.Sprintf("urn:opds-server:authors:%s", letter),
		fmt.Sprintf("Авторы на '%s'", letter),
	)
	feed.AddLink("self", fmt.Sprintf("%s/authors/%s", baseURL, letter), "application/atom+xml;profile=opds-catalog")
	feed.AddLink("up", baseURL+"/authors", "application/atom+xml;profile=opds-catalog")

	for _, author := range authors {
		title := author.FullName()
		if author.BookCount > 0 {
			title = fmt.Sprintf("%s (%d)", title, author.BookCount)
		}
		feed.AddAcquisitionEntry(
			fmt.Sprintf("urn:opds-server:author:%d", author.ID),
			title,
			fmt.Sprintf("%s/author/%d", baseURL, author.ID),
		)
	}

	s.writeOPDS(w, feed)
}

func (s *Server) handleOPDSAuthor(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	authorIDStr := chi.URLParam(r, "authorID")
	authorID, _ := strconv.ParseInt(authorIDStr, 10, 64)
	baseURL := fmt.Sprintf("/opds/%d", libID)

	author, err := s.db.GetAuthor(authorID)
	if err != nil {
		http.Error(w, "Author not found", http.StatusNotFound)
		return
	}

	books, _, err := s.db.GetBooksByAuthor(authorID, 1000, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed := opds.NewFeed(
		fmt.Sprintf("urn:opds-server:author:%d", authorID),
		author.FullName(),
	)
	feed.AddLink("self", fmt.Sprintf("%s/author/%d", baseURL, authorID), "application/atom+xml;profile=opds-catalog")
	feed.AddLink("up", baseURL+"/authors", "application/atom+xml;profile=opds-catalog")

	for _, book := range books {
		s.addBookToFeed(feed, book, libID, baseURL)
	}

	s.writeOPDS(w, feed)
}

func (s *Server) handleOPDSSeries(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	baseURL := fmt.Sprintf("/opds/%d", libID)

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}
	limit := s.config.Library.BooksPerPage
	offset := (page - 1) * limit

	series, total, err := s.db.GetSeries(libID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed := opds.NewFeed("urn:opds-server:series", "Серии")
	feed.AddLink("self", baseURL+"/series", "application/atom+xml;profile=opds-catalog")
	feed.AddLink("up", baseURL, "application/atom+xml;profile=opds-catalog")

	totalPages := (int(total) + limit - 1) / limit
	feed.AddPagination(baseURL+"/series", page, totalPages)

	for _, s := range series {
		title := s.Name
		if s.BookCount > 0 {
			title = fmt.Sprintf("%s (%d)", s.Name, s.BookCount)
		}
		feed.AddAcquisitionEntry(
			fmt.Sprintf("urn:opds-server:series:%d", s.ID),
			title,
			fmt.Sprintf("%s/series/%d", baseURL, s.ID),
		)
	}

	s.writeOPDS(w, feed)
}

func (s *Server) handleOPDSSeriesBooks(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	seriesIDStr := chi.URLParam(r, "seriesID")
	seriesID, _ := strconv.ParseInt(seriesIDStr, 10, 64)
	baseURL := fmt.Sprintf("/opds/%d", libID)

	series, err := s.db.GetSeriesByID(seriesID)
	if err != nil {
		http.Error(w, "Series not found", http.StatusNotFound)
		return
	}

	books, err := s.db.GetBooksBySeries(seriesID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed := opds.NewFeed(
		fmt.Sprintf("urn:opds-server:series:%d", seriesID),
		series.Name,
	)
	feed.AddLink("self", fmt.Sprintf("%s/series/%d", baseURL, seriesID), "application/atom+xml;profile=opds-catalog")
	feed.AddLink("up", baseURL+"/series", "application/atom+xml;profile=opds-catalog")

	for _, book := range books {
		s.addBookToFeed(feed, book, libID, baseURL)
	}

	s.writeOPDS(w, feed)
}

func (s *Server) handleOPDSGenres(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	baseURL := fmt.Sprintf("/opds/%d", libID)

	genres, err := s.db.GetTopLevelGenres()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed := opds.NewFeed("urn:opds-server:genres", "Жанры")
	feed.AddLink("self", baseURL+"/genres", "application/atom+xml;profile=opds-catalog")
	feed.AddLink("up", baseURL, "application/atom+xml;profile=opds-catalog")

	for _, genre := range genres {
		feed.AddNavEntry(
			fmt.Sprintf("urn:opds-server:genre:%d", genre.ID),
			genre.Name,
			fmt.Sprintf("%s/genres/%d", baseURL, genre.ID),
		)
	}

	s.writeOPDS(w, feed)
}

func (s *Server) handleOPDSGenreBooks(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	genreIDStr := chi.URLParam(r, "genreID")
	genreID, _ := strconv.Atoi(genreIDStr)
	baseURL := fmt.Sprintf("/opds/%d", libID)

	// Check if this is a parent genre (show subgenres) or leaf genre (show books)
	subGenres, err := s.db.GetSubGenres(genreID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	genres, _ := s.db.GetGenres()
	var genreName string
	for _, g := range genres {
		if g.ID == genreID {
			genreName = g.Name
			break
		}
	}

	feed := opds.NewFeed(
		fmt.Sprintf("urn:opds-server:genre:%d", genreID),
		genreName,
	)
	feed.AddLink("self", fmt.Sprintf("%s/genres/%d", baseURL, genreID), "application/atom+xml;profile=opds-catalog")
	feed.AddLink("up", baseURL+"/genres", "application/atom+xml;profile=opds-catalog")

	if len(subGenres) > 0 {
		// Show subgenres
		for _, genre := range subGenres {
			feed.AddNavEntry(
				fmt.Sprintf("urn:opds-server:genre:%d", genre.ID),
				genre.Name,
				fmt.Sprintf("%s/genres/%d", baseURL, genre.ID),
			)
		}
	} else {
		// Show books in this genre
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			page, _ = strconv.Atoi(p)
		}
		limit := s.config.Library.BooksPerPage
		offset := (page - 1) * limit

		books, total, err := s.db.GetBooksByGenre(genreID, libID, limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		totalPages := (int(total) + limit - 1) / limit
		feed.AddPagination(fmt.Sprintf("%s/genres/%d", baseURL, genreID), page, totalPages)

		for _, book := range books {
			s.addBookToFeed(feed, book, libID, baseURL)
		}
	}

	s.writeOPDS(w, feed)
}

func (s *Server) handleOPDSBook(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	bookIDStr := chi.URLParam(r, "bookID")
	bookID, _ := strconv.ParseInt(bookIDStr, 10, 64)
	format := chi.URLParam(r, "format")

	book, err := s.db.GetBook(bookID)
	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	lib, err := s.db.GetLibrary(libID)
	if err != nil {
		http.Error(w, "Library not found", http.StatusNotFound)
		return
	}

	// Use requested format or fall back to book's original format
	if format == "" {
		format = book.Format
	}

	bf := &bookfile.BookFile{
		LibraryPath: lib.Path,
		Archive:     book.Archive,
		File:        book.File,
		Format:      format,
	}

	reader, size, err := bf.GetReader()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get book file: %v", err), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// Set headers
	contentType := bookfile.GetMimeType(format)
	ext := bookfile.GetFileExtension(format)
	fileName := book.File + ext

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	if size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	}

	// Stream the file
	io.Copy(w, reader)
}

func (s *Server) handleOPDSCover(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	bookIDStr := chi.URLParam(r, "bookID")
	bookID, _ := strconv.ParseInt(bookIDStr, 10, 64)

	book, err := s.db.GetBook(bookID)
	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	library, err := s.db.GetLibrary(libID)
	if err != nil {
		http.Error(w, "Library not found", http.StatusNotFound)
		return
	}

	// Only FB2 files have embedded covers
	if book.Format != "fb2" {
		http.Error(w, "Cover not available", http.StatusNotFound)
		return
	}

	bf := bookfile.New(library.Path, book.Archive, book.File, book.Format)
	reader, _, err := bf.GetReader()
	if err != nil {
		http.Error(w, "Failed to read book", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	coverData, contentType, err := bookfile.ExtractFB2Cover(reader)
	if err != nil || coverData == nil {
		http.Error(w, "Cover not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(coverData)
}

func (s *Server) handleOPDSAnnotation(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	bookIDStr := chi.URLParam(r, "bookID")
	bookID, _ := strconv.ParseInt(bookIDStr, 10, 64)

	book, err := s.db.GetBook(bookID)
	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	library, err := s.db.GetLibrary(libID)
	if err != nil {
		http.Error(w, "Library not found", http.StatusNotFound)
		return
	}

	// Only FB2 files have embedded annotations
	if book.Format != "fb2" {
		http.Error(w, "Annotation not available", http.StatusNotFound)
		return
	}

	bf := bookfile.New(library.Path, book.Archive, book.File, book.Format)
	reader, _, err := bf.GetReader()
	if err != nil {
		http.Error(w, "Failed to read book", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	annotation, err := bookfile.ExtractFB2Annotation(reader)
	if err != nil || annotation == "" {
		http.Error(w, "Annotation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write([]byte(annotation))
}

func (s *Server) handleOPDSSearch(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)
	baseURL := fmt.Sprintf("/opds/%d", libID)

	query := r.URL.Query().Get("q")
	if query == "" {
		query = r.URL.Query().Get("query")
	}

	if query == "" {
		http.Error(w, "Query parameter required", http.StatusBadRequest)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}
	limit := s.config.Library.BooksPerPage
	offset := (page - 1) * limit

	books, total, err := s.db.SearchBooks(libID, query, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed := opds.NewFeed(
		"urn:opds-server:search",
		fmt.Sprintf("Поиск: %s", query),
	)
	feed.AddLink("self", fmt.Sprintf("%s/search?q=%s", baseURL, query), "application/atom+xml;profile=opds-catalog")
	feed.AddLink("up", baseURL, "application/atom+xml;profile=opds-catalog")

	totalPages := (int(total) + limit - 1) / limit
	feed.AddPagination(fmt.Sprintf("%s/search?q=%s", baseURL, query), page, totalPages)

	for _, book := range books {
		s.addBookToFeed(feed, book, libID, baseURL)
	}

	s.writeOPDS(w, feed)
}

func (s *Server) handleOpenSearch(w http.ResponseWriter, r *http.Request) {
	libID := s.getLibraryID(r)

	// Build absolute URL from request
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwdProto := r.Header.Get("X-Forwarded-Proto"); fwdProto != "" {
		scheme = fwdProto
	}
	host := r.Host
	baseURL := fmt.Sprintf("%s://%s/opds/%d", scheme, host, libID)

	w.Header().Set("Content-Type", "application/opensearchdescription+xml; charset=utf-8")
	w.Write([]byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/">
  <ShortName>opds-server</ShortName>
  <Description>Search books</Description>
  <InputEncoding>UTF-8</InputEncoding>
  <OutputEncoding>UTF-8</OutputEncoding>
  <Url type="application/atom+xml;profile=opds-catalog" template="%s/search?q={searchTerms}"/>
</OpenSearchDescription>`, baseURL)))
}

func (s *Server) addBookToFeed(feed *opds.Feed, book db.Book, libID int64, baseURL string) {
	authors, _ := s.db.GetBookAuthors(book.ID)
	var authorNames []string
	for _, a := range authors {
		authorNames = append(authorNames, a.FullName())
	}

	genres, _ := s.db.GetBookGenres(book.ID)
	var genreNames []string
	for _, g := range genres {
		genreNames = append(genreNames, g.Name)
	}

	series, _ := s.db.GetBookSeries(book.ID)
	var seriesName string
	var seqNum int
	if len(series) > 0 {
		seriesName = series[0].Name
		// TODO: get sequence number from book_series table
	}

	// Extract annotation from FB2 file
	var annotation string
	if book.Format == "fb2" {
		library, err := s.db.GetLibrary(libID)
		if err == nil {
			bf := bookfile.New(library.Path, book.Archive, book.File, book.Format)
			reader, _, err := bf.GetReader()
			if err == nil {
				annotation, _ = bookfile.ExtractFB2Annotation(reader)
				reader.Close()
			}
		}
	}

	entry := opds.BookEntry{
		ID:          book.ID,
		Title:       book.Title,
		Authors:     authorNames,
		SeriesName:  seriesName,
		SeriesNum:   seqNum,
		Genres:      genreNames,
		Language:    book.Lang,
		Format:      book.Format,
		Size:        book.Size,
		AddedAt:     book.AddedAt,
		Annotation:  annotation,
		DownloadURL: fmt.Sprintf("%s/book/%d/%s", baseURL, book.ID, book.Format),
		CoverURL:    fmt.Sprintf("%s/covers/%d/cover.jpg", baseURL, book.ID),
	}

	feed.AddBookEntry(entry, baseURL)
}
