package bookfile

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"io"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"
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
	Images []fb2Image `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 image"`
}

type fb2Annotation struct {
	Paragraphs []string `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 p"`
}

type fb2TitleInfo struct {
	Annotation fb2Annotation `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 annotation"`
	Coverpage  fb2Coverpage  `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 coverpage"`
}

type fb2Description struct {
	TitleInfo fb2TitleInfo `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 title-info"`
}

type fb2Document struct {
	XMLName     xml.Name       `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 FictionBook"`
	Description fb2Description `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 description"`
	Binaries    []fb2Binary    `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 binary"`
}

// ExtractFB2Annotation extracts the annotation/description from an FB2 file
func ExtractFB2Annotation(reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	var doc fb2Document
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charsetReader
	decoder.Strict = false

	if err := decoder.Decode(&doc); err != nil {
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
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charsetReader
	decoder.Strict = false

	if err := decoder.Decode(&doc); err != nil {
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

// charsetReader handles various character encodings in FB2 files
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
		enc, err := ianaindex.IANA.Encoding(charset)
		if err != nil {
			return input, nil
		}
		if enc == nil {
			return input, nil
		}
		return enc.NewDecoder().Reader(input), nil
	}
}
