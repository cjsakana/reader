package encoding

import (
	"io"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// DetectAndConvert detects the encoding of raw bytes and converts to UTF-8.
// Detection order: BOM check → UTF-8 → GBK → GB18030 → raw fallback.
func DetectAndConvert(data []byte) string {
	// Check UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return string(data[3:])
	}

	// Try UTF-8 directly
	s := string(data)
	if IsValidUTF8(s) {
		return s
	}

	// Try GBK
	reader := transform.NewReader(strings.NewReader(string(data)), simplifiedchinese.GBK.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err == nil {
		result := string(decoded)
		if IsValidUTF8(result) {
			return result
		}
	}

	// Try GB18030
	reader = transform.NewReader(strings.NewReader(string(data)), simplifiedchinese.GB18030.NewDecoder())
	decoded, err = io.ReadAll(reader)
	if err == nil {
		return string(decoded)
	}

	// Fallback: raw bytes as string
	return s
}

// IsValidUTF8 returns false if any rune is the Unicode replacement character (U+FFFD),
// which indicates invalid UTF-8 sequences.
func IsValidUTF8(s string) bool {
	for _, r := range s {
		if r == 0xFFFD {
			return false
		}
	}
	return true
}
