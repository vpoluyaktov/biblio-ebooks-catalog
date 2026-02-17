package parser

import (
	"fmt"
	"io"
)

// Metadata represents book metadata extracted from various formats
type Metadata struct {
	Title       string
	Authors     []string
	Language    string
	Description string
	Genres      []string
	Series      string
	SeriesIndex int
	CoverData   []byte
	CoverType   string
}

// Chapter represents a readable chapter/section in a book.
type Chapter struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// BookContent represents full content used by the web reader.
type BookContent struct {
	Title    string    `json:"title"`
	Author   string    `json:"author"`
	Format   string    `json:"format"`
	Chapters []Chapter `json:"chapters"`
}

// Parser defines the interface for book metadata parsers
type Parser interface {
	// Parse extracts metadata from a file path
	Parse(filePath string) (*Metadata, error)

	// ParseFromBytes extracts metadata from raw bytes
	ParseFromBytes(data []byte) (*Metadata, error)

	// ParseFromReader extracts metadata from an io.Reader
	ParseFromReader(reader io.Reader) (*Metadata, error)

	// Format returns the format this parser handles (e.g., "epub", "fb2")
	Format() string
}

// Registry holds registered parsers for different formats
type Registry struct {
	parsers map[string]Parser
}

// NewRegistry creates a new parser registry
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[string]Parser),
	}
}

// Register adds a parser for a specific format
func (r *Registry) Register(format string, parser Parser) {
	r.parsers[format] = parser
}

// Get returns a parser for the specified format
func (r *Registry) Get(format string) (Parser, error) {
	parser, ok := r.parsers[format]
	if !ok {
		return nil, fmt.Errorf("no parser registered for format: %s", format)
	}
	return parser, nil
}

// Parse is a convenience method to parse a file using the appropriate parser
func (r *Registry) Parse(format, filePath string) (*Metadata, error) {
	parser, err := r.Get(format)
	if err != nil {
		return nil, err
	}
	return parser.Parse(filePath)
}

// ParseFromBytes is a convenience method to parse bytes using the appropriate parser
func (r *Registry) ParseFromBytes(format string, data []byte) (*Metadata, error) {
	parser, err := r.Get(format)
	if err != nil {
		return nil, err
	}
	return parser.ParseFromBytes(data)
}

// DefaultRegistry is the global parser registry with all standard parsers registered
// Note: Now using the unified biblio-ebook-parser library
var DefaultRegistry = NewRegistry()

// Parse is a convenience function using the default registry
func Parse(format, filePath string) (*Metadata, error) {
	return DefaultRegistry.Parse(format, filePath)
}

// ParseFromBytes is a convenience function using the default registry
func ParseFromBytes(format string, data []byte) (*Metadata, error) {
	return DefaultRegistry.ParseFromBytes(format, data)
}
