package parser

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
