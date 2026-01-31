package reader

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Chapter represents a chapter in an ebook
type Chapter struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// BookContent represents the full content structure of an ebook
type BookContent struct {
	Title    string    `json:"title"`
	Author   string    `json:"author"`
	Format   string    `json:"format"`
	Chapters []Chapter `json:"chapters"`
}

// ExtractContent extracts readable content from an ebook file
func ExtractContent(reader io.ReaderAt, size int64, format string) (*BookContent, error) {
	format = strings.ToLower(format)

	switch format {
	case "epub", "epub.zip":
		return extractEPUBContent(reader, size)
	case "fb2", "fb2.zip":
		return extractFB2Content(reader, size)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// extractEPUBContent extracts content from EPUB files
func extractEPUBContent(reader io.ReaderAt, size int64) (*BookContent, error) {
	zr, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to open EPUB: %w", err)
	}

	// Find container.xml
	containerFile, err := findFileInZip(zr, "META-INF/container.xml")
	if err != nil {
		return nil, fmt.Errorf("container.xml not found: %w", err)
	}

	var container struct {
		XMLName  xml.Name `xml:"container"`
		RootFile struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}

	if err := parseXMLFromZipFile(containerFile, &container); err != nil {
		return nil, fmt.Errorf("failed to parse container.xml: %w", err)
	}

	// Find and parse package file
	packageFile, err := findFileInZip(zr, container.RootFile.FullPath)
	if err != nil {
		return nil, fmt.Errorf("package file not found: %w", err)
	}

	var pkg struct {
		XMLName  xml.Name `xml:"package"`
		Metadata struct {
			Titles   []string `xml:"title"`
			Creators []struct {
				Name string `xml:",chardata"`
			} `xml:"creator"`
		} `xml:"metadata"`
		Manifest struct {
			Items []struct {
				ID        string `xml:"id,attr"`
				Href      string `xml:"href,attr"`
				MediaType string `xml:"media-type,attr"`
			} `xml:"item"`
		} `xml:"manifest"`
		Spine struct {
			ItemRefs []struct {
				IDRef string `xml:"idref,attr"`
			} `xml:"itemref"`
		} `xml:"spine"`
	}

	if err := parseXMLFromZipFile(packageFile, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package file: %w", err)
	}

	content := &BookContent{
		Format:   "epub",
		Chapters: []Chapter{},
	}

	// Extract metadata
	if len(pkg.Metadata.Titles) > 0 {
		content.Title = pkg.Metadata.Titles[0]
	}
	if len(pkg.Metadata.Creators) > 0 {
		content.Author = pkg.Metadata.Creators[0].Name
	}

	// Build manifest map
	manifestMap := make(map[string]string)
	for _, item := range pkg.Manifest.Items {
		manifestMap[item.ID] = item.Href
	}

	// Extract chapters in spine order
	baseDir := filepath.Dir(container.RootFile.FullPath)
	for i, itemRef := range pkg.Spine.ItemRefs {
		href, ok := manifestMap[itemRef.IDRef]
		if !ok {
			continue
		}

		// Construct full path
		fullPath := filepath.Join(baseDir, href)
		chapterFile, err := findFileInZip(zr, fullPath)
		if err != nil {
			continue
		}

		rc, err := chapterFile.Open()
		if err != nil {
			continue
		}

		chapterData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		// Extract title from HTML if possible
		chapterTitle := fmt.Sprintf("Chapter %d", i+1)
		htmlContent := string(chapterData)

		// Try to extract title from h1, h2, or title tags
		if titleStart := strings.Index(htmlContent, "<title>"); titleStart != -1 {
			titleEnd := strings.Index(htmlContent[titleStart:], "</title>")
			if titleEnd != -1 {
				chapterTitle = htmlContent[titleStart+7 : titleStart+titleEnd]
			}
		} else if h1Start := strings.Index(htmlContent, "<h1"); h1Start != -1 {
			h1End := strings.Index(htmlContent[h1Start:], "</h1>")
			if h1End != -1 {
				// Find the end of opening tag
				tagEnd := strings.Index(htmlContent[h1Start:], ">")
				if tagEnd != -1 && tagEnd < h1End {
					chapterTitle = htmlContent[h1Start+tagEnd+1 : h1Start+h1End]
					chapterTitle = stripHTMLTags(chapterTitle)
				}
			}
		}

		content.Chapters = append(content.Chapters, Chapter{
			ID:      itemRef.IDRef,
			Title:   strings.TrimSpace(chapterTitle),
			Content: htmlContent,
		})
	}

	return content, nil
}

// extractFB2Content extracts content from FB2 files
func extractFB2Content(reader io.ReaderAt, size int64) (*BookContent, error) {
	// Read the entire FB2 file
	data := make([]byte, size)
	_, err := reader.ReadAt(data, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to read FB2 file: %w", err)
	}

	// Convert from windows-1251 to UTF-8 if needed
	data, err = convertToUTF8(data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert encoding: %w", err)
	}

	// Parse FB2 XML structure
	// Note: Using space prefix to match any namespace
	var fb2 struct {
		XMLName     xml.Name `xml:"FictionBook"`
		Description struct {
			TitleInfo struct {
				BookTitle string `xml:"book-title"`
				Authors   []struct {
					FirstName  string `xml:"first-name"`
					LastName   string `xml:"last-name"`
					MiddleName string `xml:"middle-name"`
				} `xml:"author"`
			} `xml:"title-info"`
		} `xml:"description"`
		Body []struct {
			Title struct {
				Paragraphs []string `xml:"p"`
			} `xml:"title"`
			Sections []fb2Section `xml:"section"`
		} `xml:"body"`
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		// Handle different character encodings
		return input, nil
	}

	if err := decoder.Decode(&fb2); err != nil {
		return nil, fmt.Errorf("failed to parse FB2 XML: %w (first 200 bytes: %s)", err, string(data[:min(200, len(data))]))
	}

	content := &BookContent{
		Format:   "fb2",
		Chapters: []Chapter{},
	}

	// Extract metadata
	content.Title = fb2.Description.TitleInfo.BookTitle
	if len(fb2.Description.TitleInfo.Authors) > 0 {
		author := fb2.Description.TitleInfo.Authors[0]
		authorName := strings.TrimSpace(author.FirstName + " " + author.MiddleName + " " + author.LastName)
		content.Author = strings.TrimSpace(authorName)
	}

	// Extract chapters from body sections
	chapterNum := 1
	for _, body := range fb2.Body {
		for _, section := range body.Sections {
			chapter := extractFB2Section(section, &chapterNum)
			if chapter.Content != "" {
				content.Chapters = append(content.Chapters, chapter)
			}
		}
	}

	return content, nil
}

type fb2Section struct {
	Title struct {
		Paragraphs []string `xml:"p"`
	} `xml:"title"`
	Paragraphs []string     `xml:"p"`
	Sections   []fb2Section `xml:"section"`
}

func extractFB2Section(section fb2Section, chapterNum *int) Chapter {
	var html strings.Builder

	// Extract title
	title := fmt.Sprintf("Chapter %d", *chapterNum)
	if len(section.Title.Paragraphs) > 0 {
		title = strings.TrimSpace(section.Title.Paragraphs[0])
		html.WriteString("<h2>")
		html.WriteString(htmlEscape(title))
		html.WriteString("</h2>\n")
	}

	// Extract paragraphs
	for _, p := range section.Paragraphs {
		html.WriteString("<p>")
		html.WriteString(htmlEscape(p))
		html.WriteString("</p>\n")
	}

	chapter := Chapter{
		ID:      fmt.Sprintf("chapter-%d", *chapterNum),
		Title:   title,
		Content: html.String(),
	}

	*chapterNum++

	// Process subsections recursively
	for _, subsection := range section.Sections {
		subChapter := extractFB2Section(subsection, chapterNum)
		if subChapter.Content != "" {
			// For now, we'll treat subsections as separate chapters
			// In a more sophisticated implementation, we might nest them
		}
	}

	return chapter
}

// Helper functions

func findFileInZip(zr *zip.Reader, path string) (*zip.File, error) {
	path = filepath.ToSlash(path)
	for _, f := range zr.File {
		if filepath.ToSlash(f.Name) == path {
			return f, nil
		}
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

func parseXMLFromZipFile(f *zip.File, v interface{}) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	return decoder.Decode(v)
}

func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// convertToUTF8 detects encoding from XML declaration and converts to UTF-8
func convertToUTF8(data []byte) ([]byte, error) {
	// Extract encoding from XML declaration
	encodingRegex := regexp.MustCompile(`encoding="([^"]+)"`)
	matches := encodingRegex.FindSubmatch(data[:min(200, len(data))])

	if len(matches) < 2 {
		// No encoding specified, assume UTF-8
		return data, nil
	}

	encoding := strings.ToLower(string(matches[1]))

	// If already UTF-8, return as-is
	if encoding == "utf-8" || encoding == "utf8" {
		return data, nil
	}

	// Handle windows-1251 (Cyrillic)
	if encoding == "windows-1251" || encoding == "cp1251" {
		decoder := charmap.Windows1251.NewDecoder()
		utf8Data, _, err := transform.Bytes(decoder, data)
		if err != nil {
			return nil, fmt.Errorf("failed to convert from windows-1251: %w", err)
		}

		// Update XML declaration to UTF-8
		utf8Data = encodingRegex.ReplaceAll(utf8Data, []byte(`encoding="UTF-8"`))
		return utf8Data, nil
	}

	// Handle other common encodings
	var decoder transform.Transformer
	switch encoding {
	case "windows-1252", "cp1252":
		decoder = charmap.Windows1252.NewDecoder()
	case "iso-8859-1", "latin1":
		decoder = charmap.ISO8859_1.NewDecoder()
	case "koi8-r":
		decoder = charmap.KOI8R.NewDecoder()
	default:
		// Unknown encoding, try to parse as-is
		return data, nil
	}

	if decoder != nil {
		utf8Data, _, err := transform.Bytes(decoder, data)
		if err != nil {
			return nil, fmt.Errorf("failed to convert from %s: %w", encoding, err)
		}

		// Update XML declaration to UTF-8
		utf8Data = encodingRegex.ReplaceAll(utf8Data, []byte(`encoding="UTF-8"`))
		return utf8Data, nil
	}

	return data, nil
}
