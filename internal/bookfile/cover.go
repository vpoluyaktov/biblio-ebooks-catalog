package bookfile

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"io"
	"strings"

	"biblio-catalog/internal/parser"
)

type fb2Document struct {
	Description fb2Description     `xml:"description"`
	Binaries    []parser.Fb2Binary `xml:"binary"`
}

type fb2Description struct {
	TitleInfo parser.Fb2TitleInfo `xml:"title-info"`
}

// ExtractFB2Annotation extracts the annotation/description from an FB2 file
func ExtractFB2Annotation(reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	var doc fb2Document
	if err := xml.Unmarshal(data, &doc); err != nil {
		return "", err
	}

	paragraphs := doc.Description.TitleInfo.Annotation.Paragraphs
	if len(paragraphs) == 0 {
		return "", nil
	}

	return strings.Join(paragraphs, "\n\n"), nil
}

func ExtractFB2Cover(reader io.Reader) ([]byte, string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}

	var doc fb2Document
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, "", err
	}

	// Find cover image reference
	var coverID string
	for _, img := range doc.Description.TitleInfo.Coverpage.Images {
		href := img.Href
		if href == "" {
			href = img.XlinkHref
		}
		if href == "" {
			href = img.LHref
		}
		if href != "" {
			// Remove leading #
			coverID = strings.TrimPrefix(href, "#")
			break
		}
	}

	if coverID == "" {
		return nil, "", nil
	}

	// Find binary data
	for _, binary := range doc.Binaries {
		if binary.ID == coverID {
			// Decode base64
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(binary.Data))
			if err != nil {
				return nil, "", err
			}

			contentType := binary.ContentType
			if contentType == "" {
				// Detect from data
				if bytes.HasPrefix(decoded, []byte{0xFF, 0xD8, 0xFF}) {
					contentType = "image/jpeg"
				} else if bytes.HasPrefix(decoded, []byte{0x89, 0x50, 0x4E, 0x47}) {
					contentType = "image/png"
				} else {
					contentType = "image/jpeg"
				}
			}

			return decoded, contentType, nil
		}
	}

	return nil, "", nil
}
