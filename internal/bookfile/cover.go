package bookfile

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"io"
	"strings"
)

type fb2Binary struct {
	ID          string `xml:"id,attr"`
	ContentType string `xml:"content-type,attr"`
	Data        string `xml:",chardata"`
}

type fb2Image struct {
	Href      string `xml:"href,attr"`
	XlinkHref string `xml:"http://www.w3.org/1999/xlink href,attr"`
	LHref     string `xml:"l:href,attr"`
}

type fb2Coverpage struct {
	Images []fb2Image `xml:"image"`
}

type fb2Annotation struct {
	Paragraphs []string `xml:"p"`
}

type fb2TitleInfo struct {
	Annotation fb2Annotation `xml:"annotation"`
	Coverpage  fb2Coverpage  `xml:"coverpage"`
}

type fb2Description struct {
	TitleInfo fb2TitleInfo `xml:"title-info"`
}

type fb2Document struct {
	Description fb2Description `xml:"description"`
	Binaries    []fb2Binary    `xml:"binary"`
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
