package bookfile

import (
	"io"

	"github.com/vpoluyaktov/biblio-ebook-parser/cover"
	"github.com/vpoluyaktov/biblio-ebook-parser/parser"
)

// ExtractFB2Cover extracts the cover image from an FB2 file
// This is a wrapper around the unified parser library
func ExtractFB2Cover(reader io.Reader) ([]byte, string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}

	// Use the parser library's fast extraction
	return parser.ExtractCoverFromReader(&bytesReaderAt{data}, int64(len(data)), "fb2")
}

// ExtractFB2Annotation extracts the annotation/description from an FB2 file
// This is a wrapper around the unified parser library
func ExtractFB2Annotation(reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	// Use the parser library's fast extraction
	return parser.ExtractAnnotationFromReader(&bytesReaderAt{data}, int64(len(data)), "fb2")
}

// GeneratePlaceholderCover creates a book cover image with title and author
// This is a wrapper around the unified parser library
func GeneratePlaceholderCover(title, author string) ([]byte, error) {
	return cover.GeneratePlaceholder(title, author)
}

// ExtractEPUBCover extracts the cover image from an EPUB file
func ExtractEPUBCover(reader io.Reader) ([]byte, string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}

	// Use the parser library's fast extraction
	return parser.ExtractCoverFromReader(&bytesReaderAt{data}, int64(len(data)), "epub")
}

// ExtractEPUBAnnotation extracts the annotation/description from an EPUB file
func ExtractEPUBAnnotation(reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	// Use the parser library's fast extraction
	return parser.ExtractAnnotationFromReader(&bytesReaderAt{data}, int64(len(data)), "epub")
}

// bytesReaderAt implements io.ReaderAt for a byte slice
type bytesReaderAt struct {
	data []byte
}

func (r *bytesReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, io.EOF
	}
	if off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n = copy(p, r.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}
