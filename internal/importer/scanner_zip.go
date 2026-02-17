package importer

import (
	"archive/zip"
	"biblio-ebooks-catalog/internal/parser"
	"io"
	"log"
	"strings"
)

// parseZipArchive extracts and parses all FB2 files from a ZIP archive
func (s *Scanner) parseZipArchive(zipPath, relPath string) []*ScannedBook {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		log.Printf("Warning: failed to open ZIP %s: %v", zipPath, err)
		return nil
	}
	defer r.Close()

	var books []*ScannedBook

	// Iterate through all files in the ZIP
	for _, f := range r.File {
		// Only process .fb2 files
		if !strings.HasSuffix(strings.ToLower(f.Name), ".fb2") {
			continue
		}

		// Open the FB2 file from the ZIP
		rc, err := f.Open()
		if err != nil {
			log.Printf("Warning: failed to open %s in %s: %v", f.Name, zipPath, err)
			continue
		}

		// Parse FB2 metadata using existing parser
		metadata, parseErr := parseFB2MetadataFromReader(rc)
		rc.Close()

		book := &ScannedBook{
			FilePath:  zipPath,
			RelPath:   relPath,
			FileName:  strings.TrimSuffix(f.Name, ".fb2"),
			Format:    "fb2.zip",
			Size:      int64(f.UncompressedSize64),
			Archive:   zipPath,
			FileInZip: f.Name,
		}

		if parseErr != nil {
			log.Printf("Warning: failed to parse %s in %s: %v", f.Name, zipPath, parseErr)
			book.ParseError = parseErr
			// Still include the book with basic file info
		} else {
			book.Metadata = metadata
		}

		books = append(books, book)
	}

	if len(books) == 0 {
		log.Printf("Warning: no FB2 files found in %s", zipPath)
	}

	return books
}

// parseFB2MetadataFromReader wraps the parser
func parseFB2MetadataFromReader(r io.Reader) (*parser.Metadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parser.ParseMetadataFromBytes(data, "fb2")
}
