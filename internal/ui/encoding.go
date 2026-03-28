package ui

import (
	"io"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

// NormalizeEncoding normalizes encoding strings for comparison.
func NormalizeEncoding(encoding string) string {
	normalized := strings.ToUpper(strings.TrimSpace(encoding))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	return normalized
}

// IsUTF8Encoding checks if the encoding is UTF-8 compatible.
func IsUTF8Encoding(encoding string) bool {
	switch NormalizeEncoding(encoding) {
	case "", "UTF-8", "UTF8":
		return true
	default:
		return false
	}
}

// ConvertToUTF8 converts input bytes to UTF-8 based on the specified encoding.
// Currently supports ISO-8859-1 as a basic fallback.
func ConvertToUTF8(encoding string, data []byte) []byte {
	if IsUTF8Encoding(encoding) {
		return data
	}
	if cm := GetCharmap(encoding); cm != nil {
		decoded, err := cm.NewDecoder().Bytes(data)
		if err == nil {
			return decoded
		}
	}
	return data
}

// WrapEncodingWriter wraps a writer to convert output to UTF-8 before writing.
func WrapEncodingWriter(w io.Writer, encoding string) io.Writer {
	if IsUTF8Encoding(encoding) {
		return w
	}
	return &encodingWriter{w: w, encoding: encoding}
}

// GetCharmap returns the Charmap for the given encoding.
func GetCharmap(encoding string) *charmap.Charmap {
	switch NormalizeEncoding(encoding) {
	case "ISO-8859-1", "ISO8859-1", "LATIN1":
		return charmap.ISO8859_1
	case "ISO-8859-2", "ISO8859-2", "LATIN2":
		return charmap.ISO8859_2
	case "ISO-8859-15", "ISO8859-15", "LATIN9":
		return charmap.ISO8859_15
	case "WINDOWS-1252", "CP1252":
		return charmap.Windows1252
	case "WINDOWS-1251", "CP1251":
		return charmap.Windows1251
	case "KOI8-R", "KOI8R":
		return charmap.KOI8R
	case "KOI8-U", "KOI8U":
		return charmap.KOI8U
	default:
		return nil
	}
}

// encodingWriter converts output to UTF-8 before writing.
type encodingWriter struct {
	w        io.Writer
	encoding string
}

func (ew *encodingWriter) Write(p []byte) (int, error) {
	if IsUTF8Encoding(ew.encoding) {
		return ew.w.Write(p)
	}
	converted := ConvertToUTF8(ew.encoding, p)
	_, err := ew.w.Write(converted)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
