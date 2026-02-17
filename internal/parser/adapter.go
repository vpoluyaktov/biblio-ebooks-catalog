package parser

import (
	"bytes"
	"fmt"
	"io"

	_ "github.com/vpoluyaktov/biblio-ebook-parser/formats" // Register parsers
	ebookparser "github.com/vpoluyaktov/biblio-ebook-parser/parser"
	"github.com/vpoluyaktov/biblio-ebook-parser/renderer/html"
)

// ExtractContent extracts reader content for supported book formats using the unified parser.
func ExtractContent(reader io.ReaderAt, size int64, format string) (*BookContent, error) {
	// Parse the book using the unified parser
	parser, err := ebookparser.GetParser(format)
	if err != nil {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	book, err := parser.ParseReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to parse book: %w", err)
	}

	// Render to HTML for web reader
	renderer := html.NewRenderer(html.Config{
		PreserveStructure: false,
	})

	content, err := renderer.RenderContent(book)
	if err != nil {
		return nil, fmt.Errorf("failed to render content: %w", err)
	}

	// Convert to our BookContent format
	htmlContent, ok := content.(*html.BookContent)
	if !ok {
		return nil, fmt.Errorf("unexpected content type")
	}

	result := &BookContent{
		Title:    htmlContent.Title,
		Author:   htmlContent.Author,
		Format:   htmlContent.Format,
		Chapters: make([]Chapter, len(htmlContent.Chapters)),
	}

	for i, ch := range htmlContent.Chapters {
		result.Chapters[i] = Chapter{
			ID:      ch.ID,
			Title:   ch.Title,
			Content: ch.Content,
		}
	}

	return result, nil
}

// ParseMetadata extracts metadata using the unified parser
func ParseMetadata(reader io.ReaderAt, size int64, format string) (*Metadata, error) {
	parser, err := ebookparser.GetParser(format)
	if err != nil {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	book, err := parser.ParseReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to parse book: %w", err)
	}

	// Convert to our Metadata format
	metadata := &Metadata{
		Title:       book.Metadata.Title,
		Language:    book.Metadata.Language,
		Description: book.Metadata.Description,
		Genres:      book.Metadata.Genres,
		Series:      book.Metadata.Series,
		SeriesIndex: book.Metadata.SeriesIndex,
		CoverData:   book.Metadata.CoverData,
		CoverType:   book.Metadata.CoverType,
	}

	// Convert authors
	metadata.Authors = make([]string, len(book.Metadata.Authors))
	for i, author := range book.Metadata.Authors {
		metadata.Authors[i] = author.FullName()
	}

	return metadata, nil
}

// ParseMetadataFromFile extracts metadata from a file path
func ParseMetadataFromFile(filePath string, format string) (*Metadata, error) {
	parser, err := ebookparser.GetParser(format)
	if err != nil {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	book, err := parser.Parse(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse book: %w", err)
	}

	// Convert to our Metadata format
	metadata := &Metadata{
		Title:       book.Metadata.Title,
		Language:    book.Metadata.Language,
		Description: book.Metadata.Description,
		Genres:      book.Metadata.Genres,
		Series:      book.Metadata.Series,
		SeriesIndex: book.Metadata.SeriesIndex,
		CoverData:   book.Metadata.CoverData,
		CoverType:   book.Metadata.CoverType,
	}

	// Convert authors
	metadata.Authors = make([]string, len(book.Metadata.Authors))
	for i, author := range book.Metadata.Authors {
		metadata.Authors[i] = author.FullName()
	}

	return metadata, nil
}

// ParseMetadataFromBytes extracts metadata from raw bytes
func ParseMetadataFromBytes(data []byte, format string) (*Metadata, error) {
	parser, err := ebookparser.GetParser(format)
	if err != nil {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	// Use ParseReader with a bytes.Reader (which implements io.ReaderAt)
	reader := bytes.NewReader(data)
	book, err := parser.ParseReader(reader, int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse book: %w", err)
	}

	// Convert to our Metadata format
	metadata := &Metadata{
		Title:       book.Metadata.Title,
		Language:    book.Metadata.Language,
		Description: book.Metadata.Description,
		Genres:      book.Metadata.Genres,
		Series:      book.Metadata.Series,
		SeriesIndex: book.Metadata.SeriesIndex,
		CoverData:   book.Metadata.CoverData,
		CoverType:   book.Metadata.CoverType,
	}

	// Convert authors
	metadata.Authors = make([]string, len(book.Metadata.Authors))
	for i, author := range book.Metadata.Authors {
		metadata.Authors[i] = author.FullName()
	}

	return metadata, nil
}
