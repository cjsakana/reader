package chunker

import (
	"strings"
	"unicode/utf8"
)

// ChunkConfig controls how text is split into overlapping chunks.
type ChunkConfig struct {
	MaxChunkSize    int  // 最大块大小（字符数），推荐 500-800
	MinChunkSize    int  // 最小块大小，推荐 300
	OverlapSize     int  // 重叠大小，推荐 50-100
	PreferParagraph bool // 优先按段落分割
}

// Chunk is a segment of text with its character offsets.
type Chunk struct {
	Text     string // 块文本
	StartChar int   // 在原文中的起始字符位置
	EndChar   int   // 在原文中的结束字符位置（不包含）
}

// DefaultConfig returns the recommended chunking configuration.
func DefaultConfig() ChunkConfig {
	return ChunkConfig{
		MaxChunkSize:    700,
		MinChunkSize:    300,
		OverlapSize:     80,
		PreferParagraph: true,
	}
}

// Split divides text into overlapping chunks. When PreferParagraph is true,
// chunks are split at paragraph boundaries (double newline). If a single
// paragraph exceeds MaxChunkSize, it falls back to sentence-level splitting.
func Split(text string, cfg ChunkConfig) []Chunk {
	if cfg.MaxChunkSize <= 0 {
		cfg.MaxChunkSize = 700
	}
	if cfg.OverlapSize <= 0 {
		cfg.OverlapSize = 80
	}

	// Normalize line endings.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	if cfg.PreferParagraph {
		return splitByParagraph(text, cfg)
	}
	return splitBySize(text, cfg)
}

// splitByParagraph splits text at paragraph boundaries, falling back to
// sentence splitting for oversized paragraphs.
func splitByParagraph(text string, cfg ChunkConfig) []Chunk {
	paragraphs := strings.Split(text, "\n\n")
	var chunks []Chunk

	charPos := 0 // current character position in the original text
	accumulated := ""

	for _, para := range paragraphs {
		if len(accumulated)+len(para) <= cfg.MaxChunkSize {
			if accumulated == "" {
				accumulated = para
			} else {
				accumulated += "\n\n" + para
			}
			continue
		}

		// Current paragraph would exceed the limit.
		if len(accumulated) > 0 {
			// Emit what we've accumulated.
			chunks = append(chunks, Chunk{
				Text:      accumulated,
				StartChar: charPos,
				EndChar:   charPos + utf8.RuneCountInString(accumulated),
			})
			charPos += utf8.RuneCountInString(accumulated) + 2 // +2 for the \n\n before next para
			accumulated = ""
		}

		// If the paragraph itself is too large, sentence-split it.
		if len(para) > cfg.MaxChunkSize {
			for _, ch := range splitBySize(para, cfg) {
				ch.StartChar += charPos
				ch.EndChar += charPos
				chunks = append(chunks, ch)
			}
			charPos += utf8.RuneCountInString(para) + 2
		} else {
			accumulated = para
		}
	}

	// Emit remaining accumulation.
	if len(accumulated) > 0 {
		chunks = append(chunks, Chunk{
			Text:      accumulated,
			StartChar: charPos,
			EndChar:   charPos + utf8.RuneCountInString(accumulated),
		})
	}

	return filterSmall(chunks, cfg.MinChunkSize)
}

// splitBySize splits text into fixed-size overlapping chunks, preferring
// sentence boundaries (Chinese 。！？!? and newlines).
func splitBySize(text string, cfg ChunkConfig) []Chunk {
	if len(text) <= cfg.MaxChunkSize {
		return filterSmall([]Chunk{{
			Text:      text,
			StartChar: 0,
			EndChar:   utf8.RuneCountInString(text),
		}}, cfg.MinChunkSize)
	}

	// Sentence terminators: Chinese and English.
	terminators := func(r rune) bool {
		switch r {
		case '。', '！', '？', '!', '?', '\n', '…':
			return true
		}
		return false
	}

	var chunks []Chunk
	runeText := []rune(text)
	start := 0
	end := 0

	for end < len(runeText) {
		remaining := len(runeText) - start
		if remaining <= cfg.MaxChunkSize {
			chunk := Chunk{
				Text:      string(runeText[start:]),
				StartChar: start,
				EndChar:   len(runeText),
			}
			chunks = append(chunks, chunk)
			break
		}

		// Target end = start + MaxChunkSize, then walk back to a terminator.
		targetEnd := start + cfg.MaxChunkSize
		if targetEnd > len(runeText) {
			targetEnd = len(runeText)
		}

		// Walk back from targetEnd to find a good break point.
		found := false
		for i := targetEnd - 1; i > start+cfg.MaxChunkSize/2; i-- {
			if terminators(runeText[i]) {
				end = i + 1 // include the terminator
				found = true
				break
			}
		}
		if !found {
			end = targetEnd
		}

		chunks = append(chunks, Chunk{
			Text:      string(runeText[start:end]),
			StartChar: start,
			EndChar:   end,
		})

		// Advance with overlap.
		overlap := cfg.OverlapSize
		if end-start-overlap < cfg.MinChunkSize && len(chunks) > 0 {
			// Don't let overlap create a tiny final chunk.
			overlap = 0
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
		if start >= end {
			break
		}
	}

	return filterSmall(chunks, cfg.MinChunkSize)
}

// filterSmall removes chunks below the minimum size, merging them into
// adjacent chunks when possible.
func filterSmall(chunks []Chunk, minSize int) []Chunk {
	if minSize <= 0 || len(chunks) <= 1 {
		return chunks
	}

	var result []Chunk
	for _, ch := range chunks {
		if len(ch.Text) < minSize && len(result) > 0 {
			// Merge with previous chunk.
			last := &result[len(result)-1]
			last.Text += ch.Text
			last.EndChar = ch.EndChar
		} else {
			result = append(result, ch)
		}
	}
	return result
}
