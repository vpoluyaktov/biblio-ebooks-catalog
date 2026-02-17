package parser

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"
)

type fb2ContentSection struct {
	Title struct {
		Paragraphs []fb2ContentParagraph `xml:"p"`
	} `xml:"title"`
	Paragraphs []fb2ContentParagraph `xml:"p"`
	Epigraphs  []fb2ContentEpigraph  `xml:"epigraph"`
	Sections   []fb2ContentSection   `xml:"section"`
}

type fb2ContentParagraph struct {
	Content string `xml:",innerxml"`
}

type fb2ContentEpigraph struct {
	Paragraphs []fb2ContentParagraph `xml:"p"`
}

type Fb2TitleInfo struct {
	Coverpage  Fb2Coverpage  `xml:"coverpage"`
	Annotation Fb2Annotation `xml:"annotation"`
}

type Fb2Annotation struct {
	Paragraphs []string `xml:"p"`
}

type Fb2Coverpage struct {
	Images []Fb2Image `xml:"image"`
}

type Fb2Image struct {
	Href      string `xml:"href,attr"`
	XlinkHref string `xml:"http://www.w3.org/1999/xlink href,attr"`
	LHref     string `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 href,attr"`
}

type Fb2Binary struct {
	ID          string `xml:"id,attr"`
	ContentType string `xml:"content-type,attr"`
	Data        string `xml:",chardata"`
}

type fb2MetadataDocument struct {
	XMLName     xml.Name `xml:"FictionBook"`
	Description struct {
		TitleInfo struct {
			Fb2TitleInfo
			Author struct {
				FirstName  string `xml:"first-name"`
				LastName   string `xml:"last-name"`
				MiddleName string `xml:"middle-name"`
			} `xml:"author"`
			BookTitle string   `xml:"book-title"`
			Genres    []string `xml:"genre"`
			Lang      string   `xml:"lang"`
			Sequence  struct {
				Name   string `xml:"name,attr"`
				Number string `xml:"number,attr"`
			} `xml:"sequence"`
		} `xml:"title-info"`
	} `xml:"description"`
	Binaries []Fb2Binary `xml:"binary"`
}

func ParseFB2Metadata(filePath string) (*EPUBMetadata, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return parseFB2Metadata(f)
}

func ParseFB2MetadataFromZip(zipPath string) (*EPUBMetadata, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP: %w", err)
	}
	defer r.Close()

	var fb2File *zip.File
	for _, f := range r.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".fb2") {
			fb2File = f
			break
		}
	}

	if fb2File == nil {
		return nil, fmt.Errorf("no FB2 file found in archive")
	}

	rc, err := fb2File.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open FB2 file: %w", err)
	}
	defer rc.Close()

	return parseFB2Metadata(rc)
}

func ParseFB2MetadataFromBytes(data []byte) (*EPUBMetadata, error) {
	return parseFB2MetadataFromBytes(data)
}

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	charset = strings.ToLower(charset)

	switch charset {
	case "windows-1251":
		return charmap.Windows1251.NewDecoder().Reader(input), nil
	case "windows-1252":
		return charmap.Windows1252.NewDecoder().Reader(input), nil
	case "iso-8859-1", "latin1":
		return charmap.ISO8859_1.NewDecoder().Reader(input), nil
	case "koi8-r":
		return charmap.KOI8R.NewDecoder().Reader(input), nil
	case "koi8-u":
		return charmap.KOI8U.NewDecoder().Reader(input), nil
	case "utf-8", "":
		return input, nil
	case "utf-16", "utf-16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder().Reader(input), nil
	case "utf-16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder().Reader(input), nil
	default:
		return input, nil
	}
}

func sanitizeFB2XML(data []byte) []byte {
	if !utf8.Valid(data) {
		data = fixInvalidUTF8(data)
	}

	data = removeIllegalXMLChars(data)
	data = fixUnescapedAmpersands(data)
	data = fixMalformedTags(data)

	return data
}

func fixInvalidUTF8(data []byte) []byte {
	result := make([]byte, 0, len(data))
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			if data[0] >= 0x80 {
				decoded := charmap.Windows1251.DecodeByte(data[0])
				result = utf8.AppendRune(result, decoded)
			} else {
				result = append(result, ' ')
			}
			data = data[1:]
		} else {
			result = utf8.AppendRune(result, r)
			data = data[size:]
		}
	}
	return result
}

func removeIllegalXMLChars(data []byte) []byte {
	result := make([]byte, 0, len(data))
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == 0x9 || r == 0xA || r == 0xD || (r >= 0x20 && r <= 0xD7FF) || (r >= 0xE000 && r <= 0xFFFD) || (r >= 0x10000 && r <= 0x10FFFF) {
			result = utf8.AppendRune(result, r)
		} else {
			result = append(result, ' ')
		}
		data = data[size:]
	}
	return result
}

func fixUnescapedAmpersands(data []byte) []byte {
	result := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		if data[i] == '&' {
			// Check if this is a valid entity
			remaining := string(data[i:])
			if strings.HasPrefix(remaining, "&amp;") ||
				strings.HasPrefix(remaining, "&lt;") ||
				strings.HasPrefix(remaining, "&gt;") ||
				strings.HasPrefix(remaining, "&quot;") ||
				strings.HasPrefix(remaining, "&apos;") ||
				regexp.MustCompile(`^&#[0-9]+;`).MatchString(remaining) ||
				regexp.MustCompile(`^&#x[0-9a-fA-F]+;`).MatchString(remaining) {
				// Valid entity, keep as-is
				result = append(result, data[i])
			} else {
				// Invalid/unescaped ampersand, escape it
				result = append(result, []byte("&amp;")...)
				i++
				continue
			}
		} else {
			result = append(result, data[i])
		}
		i++
	}
	return result
}

func fixMalformedTags(data []byte) []byte {
	// Fix tags starting with numbers, dots, or dashes
	reInvalidTagStart := regexp.MustCompile(`<([0-9]|\.\.\.|--?[^a-zA-Z>])`)
	data = reInvalidTagStart.ReplaceAllFunc(data, func(match []byte) []byte {
		return append([]byte("&lt;"), match[1:]...)
	})

	// Fix unescaped < followed by non-ASCII characters (e.g., Cyrillic text)
	// Valid XML element names must start with a letter (A-Z, a-z), underscore, or colon
	// This pattern matches < followed by any non-ASCII byte (0x80+) which indicates
	// text content that was not properly escaped
	result := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		if data[i] == '<' {
			// Check if this is a valid XML tag start
			if i+1 >= len(data) {
				// Bare < at end of file
				result = append(result, []byte("&lt;")...)
				i++
				continue
			}
			nextByte := data[i+1]
			// Valid tag starts: a-z, A-Z, /, !, ?, _
			isValidTagStart := (nextByte >= 'a' && nextByte <= 'z') ||
				(nextByte >= 'A' && nextByte <= 'Z') ||
				nextByte == '/' || nextByte == '!' || nextByte == '?' || nextByte == '_'

			if !isValidTagStart {
				// Invalid tag start - escape the <
				result = append(result, []byte("&lt;")...)
				i++
				continue
			}
		}
		result = append(result, data[i])
		i++
	}

	return result
}

func parseSeriesNumber(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	if n, err := strconv.Atoi(s); err == nil {
		if n > 0 {
			return n
		}
		return 1
	}

	re := regexp.MustCompile(`^(\d+)`)
	if matches := re.FindStringSubmatch(s); len(matches) > 1 {
		if n, err := strconv.Atoi(matches[1]); err == nil && n > 0 {
			return n
		}
	}

	return 1
}

func parseFB2Metadata(r io.Reader) (*EPUBMetadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read FB2: %w", err)
	}

	return parseFB2MetadataFromBytes(data)
}

func parseFB2MetadataFromBytes(data []byte) (*EPUBMetadata, error) {
	sanitizedData := sanitizeFB2XML(data)

	var fb2 fb2MetadataDocument
	decoder := xml.NewDecoder(bytes.NewReader(sanitizedData))
	decoder.CharsetReader = charsetReader
	decoder.Strict = false

	if err := decoder.Decode(&fb2); err != nil {
		decoder2 := xml.NewDecoder(bytes.NewReader(data))
		decoder2.CharsetReader = charsetReader
		decoder2.Strict = false

		if err2 := decoder2.Decode(&fb2); err2 != nil {
			return nil, fmt.Errorf("failed to parse FB2: %w", err)
		}
	}

	annotation := strings.Join(fb2.Description.TitleInfo.Annotation.Paragraphs, "\n\n")
	seriesIndex := parseSeriesNumber(fb2.Description.TitleInfo.Sequence.Number)

	metadata := &EPUBMetadata{
		Title:       strings.TrimSpace(fb2.Description.TitleInfo.BookTitle),
		Language:    strings.TrimSpace(fb2.Description.TitleInfo.Lang),
		Description: strings.TrimSpace(annotation),
		Series:      strings.TrimSpace(fb2.Description.TitleInfo.Sequence.Name),
		SeriesIndex: seriesIndex,
		Genres:      fb2.Description.TitleInfo.Genres,
	}

	author := Author{
		FirstName:  strings.TrimSpace(fb2.Description.TitleInfo.Author.FirstName),
		LastName:   strings.TrimSpace(fb2.Description.TitleInfo.Author.LastName),
		MiddleName: strings.TrimSpace(fb2.Description.TitleInfo.Author.MiddleName),
	}
	if author.FirstName != "" || author.LastName != "" {
		metadata.Authors = []Author{author}
	}

	var coverID string
	for _, img := range fb2.Description.TitleInfo.Coverpage.Images {
		href := img.Href
		if href == "" {
			href = img.XlinkHref
		}
		if href == "" {
			href = img.LHref
		}
		if href != "" {
			coverID = strings.TrimPrefix(href, "#")
			break
		}
	}

	if coverID != "" {
		for _, binary := range fb2.Binaries {
			if binary.ID == coverID {
				decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(binary.Data))
				if err == nil {
					metadata.CoverData = decoded
					metadata.CoverType = binary.ContentType
					if metadata.CoverType == "" {
						if bytes.HasPrefix(decoded, []byte{0xFF, 0xD8, 0xFF}) {
							metadata.CoverType = "image/jpeg"
						} else if bytes.HasPrefix(decoded, []byte{0x89, 0x50, 0x4E, 0x47}) {
							metadata.CoverType = "image/png"
						} else {
							metadata.CoverType = "image/jpeg"
						}
					}
				}
				break
			}
		}
	}

	return metadata, nil
}

func extractFB2Content(reader io.ReaderAt, size int64) (*BookContent, error) {
	data := make([]byte, size)
	_, err := reader.ReadAt(data, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to read FB2 file: %w", err)
	}

	contentStr := string(data)
	contentStr = regexp.MustCompile(`xmlns[^=]*="[^"]*"`).ReplaceAllString(contentStr, "")
	contentStr = regexp.MustCompile(`<[a-zA-Z]+:`).ReplaceAllString(contentStr, "<")
	contentStr = regexp.MustCompile(`</[a-zA-Z]+:`).ReplaceAllString(contentStr, "</")

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
			Name     string              `xml:"name,attr"`
			Sections []fb2ContentSection `xml:"section"`
		} `xml:"body"`
	}

	decoder := xml.NewDecoder(bytes.NewReader([]byte(contentStr)))
	decoder.Strict = false
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		enc, err := ianaindex.IANA.Encoding(charset)
		if err != nil {
			return nil, fmt.Errorf("unsupported charset %q: %w", charset, err)
		}
		if enc == nil {
			return input, nil
		}
		return enc.NewDecoder().Reader(input), nil
	}

	if err := decoder.Decode(&fb2); err != nil {
		return nil, fmt.Errorf("failed to parse FB2 XML: %w", err)
	}

	content := &BookContent{
		Format:   "fb2",
		Title:    strings.TrimSpace(fb2.Description.TitleInfo.BookTitle),
		Chapters: []Chapter{},
	}
	if len(fb2.Description.TitleInfo.Authors) > 0 {
		author := fb2.Description.TitleInfo.Authors[0]
		content.Author = strings.TrimSpace(strings.TrimSpace(author.FirstName + " " + author.MiddleName + " " + author.LastName))
	}

	chapterNum := 1
	for _, body := range fb2.Body {
		if body.Name == "notes" || body.Name == "comments" {
			continue
		}
		for _, section := range body.Sections {
			extractFB2SectionContentRecursive(section, &chapterNum, content)
		}
	}

	return content, nil
}

func extractFB2SectionContentRecursive(section fb2ContentSection, chapterNum *int, content *BookContent) {
	var html strings.Builder
	title := fmt.Sprintf("Chapter %d", *chapterNum)
	if len(section.Title.Paragraphs) > 0 {
		title = strings.TrimSpace(stripFB2TagsForContent(section.Title.Paragraphs[0].Content))
		html.WriteString("<h2>")
		html.WriteString(htmlEscapeForContent(title))
		html.WriteString("</h2>\n")
	}

	for _, epigraph := range section.Epigraphs {
		html.WriteString("<blockquote class=\"epigraph\">\n")
		for _, p := range epigraph.Paragraphs {
			html.WriteString("<p>")
			html.WriteString(convertFB2ToHTMLForContent(p.Content))
			html.WriteString("</p>\n")
		}
		html.WriteString("</blockquote>\n")
	}

	for _, p := range section.Paragraphs {
		html.WriteString("<p>")
		html.WriteString(convertFB2ToHTMLForContent(p.Content))
		html.WriteString("</p>\n")
	}

	chapter := Chapter{
		ID:      fmt.Sprintf("chapter-%d", *chapterNum),
		Title:   title,
		Content: html.String(),
	}
	*chapterNum++

	if chapter.Content != "" {
		content.Chapters = append(content.Chapters, chapter)
	}

	// Recursively process subsections
	for _, subsection := range section.Sections {
		extractFB2SectionContentRecursive(subsection, chapterNum, content)
	}
}

func convertFB2ToHTMLForContent(content string) string {
	content = strings.ReplaceAll(content, "<emphasis>", "<em>")
	content = strings.ReplaceAll(content, "</emphasis>", "</em>")
	content = strings.ReplaceAll(content, "<strikethrough>", "<del>")
	content = strings.ReplaceAll(content, "</strikethrough>", "</del>")
	content = regexp.MustCompile(`<empty-line\s*/>`).ReplaceAllString(content, "<br/>")
	content = regexp.MustCompile(`<empty-line></empty-line>`).ReplaceAllString(content, "<br/>")
	content = regexp.MustCompile(`<a[^>]*l:href="[^"]*"[^>]*>`).ReplaceAllString(content, "")
	content = strings.ReplaceAll(content, "</a>", "")
	return content
}

func stripFB2TagsForContent(content string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return re.ReplaceAllString(content, "")
}

func htmlEscapeForContent(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// FB2Parser implements the Parser interface for FB2 files
type FB2Parser struct{}

// Parse extracts metadata from an FB2 file
func (p *FB2Parser) Parse(filePath string) (*Metadata, error) {
	fb2Meta, err := ParseFB2Metadata(filePath)
	if err != nil {
		return nil, err
	}
	return epubMetadataToMetadata(fb2Meta), nil
}

// ParseFromBytes extracts metadata from FB2 bytes
func (p *FB2Parser) ParseFromBytes(data []byte) (*Metadata, error) {
	fb2Meta, err := ParseFB2MetadataFromBytes(data)
	if err != nil {
		return nil, err
	}
	return epubMetadataToMetadata(fb2Meta), nil
}

// ParseFromReader extracts metadata from an FB2 reader
func (p *FB2Parser) ParseFromReader(reader io.Reader) (*Metadata, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return p.ParseFromBytes(data)
}

// Format returns the format this parser handles
func (p *FB2Parser) Format() string {
	return "fb2"
}
