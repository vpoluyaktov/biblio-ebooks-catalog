package opds

import (
	"encoding/xml"
	"fmt"
	"time"
)

const (
	AtomNS     = "http://www.w3.org/2005/Atom"
	DCNS       = "http://purl.org/dc/terms/"
	OPDSNS     = "http://opds-spec.org/2010/catalog"
	OpenSearch = "http://a9.com/-/spec/opensearch/1.1/"
)

type Feed struct {
	XMLName   xml.Name `xml:"feed"`
	Xmlns     string   `xml:"xmlns,attr"`
	XmlnsDC   string   `xml:"xmlns:dc,attr,omitempty"`
	XmlnsOS   string   `xml:"xmlns:os,attr,omitempty"`
	XmlnsOPDS string   `xml:"xmlns:opds,attr,omitempty"`

	ID      string    `xml:"id"`
	Title   string    `xml:"title"`
	Updated time.Time `xml:"updated"`
	Icon    string    `xml:"icon,omitempty"`
	Author  *Author   `xml:"author,omitempty"`
	Links   []Link    `xml:"link"`
	Entries []Entry   `xml:"entry"`
}

type Author struct {
	Name string `xml:"name"`
	URI  string `xml:"uri,omitempty"`
}

type Link struct {
	Rel         string `xml:"rel,attr,omitempty"`
	Href        string `xml:"href,attr"`
	Type        string `xml:"type,attr,omitempty"`
	Title       string `xml:"title,attr,omitempty"`
	FacetGroup  string `xml:"opds:facetGroup,attr,omitempty"`
	ActiveFacet bool   `xml:"opds:activeFacet,attr,omitempty"`
}

type Entry struct {
	ID         string        `xml:"id"`
	Title      string        `xml:"title"`
	Updated    time.Time     `xml:"updated"`
	Content    *Content      `xml:"content,omitempty"`
	Summary    string        `xml:"summary,omitempty"`
	Authors    []EntryAuthor `xml:"author,omitempty"`
	Links      []Link        `xml:"link"`
	Categories []Category    `xml:"category,omitempty"`
	SeriesName string        `xml:"series_name,omitempty"`
	SeriesNum  int           `xml:"series_num,omitempty"`
	Language   string        `xml:"dc:language,omitempty"`
	Format     string        `xml:"dc:format,omitempty"`
	Issued     string        `xml:"dc:issued,omitempty"`
	Extent     string        `xml:"dc:extent,omitempty"`
}

type EntryAuthor struct {
	Name string `xml:"name"`
}

type Content struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",chardata"`
}

type Category struct {
	Term  string `xml:"term,attr"`
	Label string `xml:"label,attr,omitempty"`
}

func NewFeed(id, title string) *Feed {
	return &Feed{
		Xmlns:     AtomNS,
		XmlnsDC:   DCNS,
		XmlnsOS:   OpenSearch,
		XmlnsOPDS: OPDSNS,
		ID:        id,
		Title:     title,
		Updated:   time.Now().UTC(),
		Icon:      "/static/img/logo.svg",
	}
}

func (f *Feed) AddLink(rel, href, linkType string) {
	f.Links = append(f.Links, Link{
		Rel:  rel,
		Href: href,
		Type: linkType,
	})
}

func (f *Feed) AddNavEntry(id, title, href string) {
	f.Entries = append(f.Entries, Entry{
		ID:      id,
		Title:   title,
		Updated: time.Now().UTC(),
		Links: []Link{{
			Href: href,
			Type: "application/atom+xml;profile=opds-catalog;kind=navigation",
			Rel:  "subsection",
		}},
	})
}

func (f *Feed) AddAcquisitionEntry(id, title, href string) {
	f.Entries = append(f.Entries, Entry{
		ID:      id,
		Title:   title,
		Updated: time.Now().UTC(),
		Links: []Link{{
			Href: href,
			Type: "application/atom+xml;profile=opds-catalog;kind=acquisition",
			Rel:  "subsection",
		}},
	})
}

func (f *Feed) ToXML() ([]byte, error) {
	output, err := xml.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), output...), nil
}

type BookEntry struct {
	ID          int64
	Title       string
	Authors     []string
	SeriesName  string
	SeriesNum   int
	Genres      []string
	Language    string
	Format      string
	Size        int64
	AddedAt     time.Time
	Annotation  string
	DownloadURL string
	CoverURL    string
}

func (f *Feed) AddBookEntry(book BookEntry, baseURL string) {
	entry := Entry{
		ID:         fmt.Sprintf("urn:book:%d", book.ID),
		Title:      book.Title,
		Updated:    book.AddedAt,
		SeriesName: book.SeriesName,
		SeriesNum:  book.SeriesNum,
		Language:   book.Language,
		Format:     book.Format,
		Extent:     formatSize(book.Size),
	}

	if book.SeriesName != "" && book.SeriesNum > 0 {
		entry.Title = fmt.Sprintf("%s (%s [%d])", book.Title, book.SeriesName, book.SeriesNum)
	}

	for _, author := range book.Authors {
		entry.Authors = append(entry.Authors, EntryAuthor{Name: author})
	}

	for _, genre := range book.Genres {
		entry.Categories = append(entry.Categories, Category{Term: genre, Label: genre})
	}

	if book.Annotation != "" {
		entry.Content = &Content{Type: "text/html", Value: book.Annotation}
	}

	// Download link
	entry.Links = append(entry.Links, Link{
		Rel:  "http://opds-spec.org/acquisition/open-access",
		Href: book.DownloadURL,
		Type: getMimeType(book.Format),
	})

	// Cover links
	if book.CoverURL != "" {
		entry.Links = append(entry.Links, Link{
			Rel:  "http://opds-spec.org/image",
			Href: book.CoverURL,
			Type: "image/jpeg",
		})
		entry.Links = append(entry.Links, Link{
			Rel:  "http://opds-spec.org/image/thumbnail",
			Href: book.CoverURL,
			Type: "image/jpeg",
		})
	}

	f.Entries = append(f.Entries, entry)
}

func formatSize(bytes int64) string {
	if bytes == 0 {
		return ""
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMG"[exp])
}

func getMimeType(format string) string {
	switch format {
	case "fb2", "fb2.zip":
		return "application/x-fictionbook+xml"
	case "epub", "epub.zip":
		return "application/epub+zip"
	case "mobi":
		return "application/x-mobipocket-ebook"
	case "azw3":
		return "application/x-mobi8-ebook"
	case "pdf", "pdf.zip":
		return "application/pdf"
	case "djvu", "djvu.zip":
		return "image/vnd.djvu"
	default:
		return "application/octet-stream"
	}
}

func (f *Feed) AddPagination(baseURL string, page, totalPages int) {
	if page > 1 {
		f.AddLink("first", fmt.Sprintf("%s?page=1", baseURL), "application/atom+xml;profile=opds-catalog")
		f.AddLink("previous", fmt.Sprintf("%s?page=%d", baseURL, page-1), "application/atom+xml;profile=opds-catalog")
	}
	if page < totalPages {
		f.AddLink("next", fmt.Sprintf("%s?page=%d", baseURL, page+1), "application/atom+xml;profile=opds-catalog")
		f.AddLink("last", fmt.Sprintf("%s?page=%d", baseURL, totalPages), "application/atom+xml;profile=opds-catalog")
	}
}
