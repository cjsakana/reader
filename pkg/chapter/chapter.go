package chapter

import (
	"regexp"
	"strings"
)

// Pattern matches common Chinese chapter titles:
// "第X章/节/回/卷/篇/集/部", preface, epilogue, prologue, etc.
var Pattern = regexp.MustCompile(
	`(?m)^\s*(第[零一二三四五六七八九十百千万\d]+[章节回卷篇集部]|序言|楔子|前言|尾声|后记|番外|附录|[Pp]rologue|[Ee]pilogue).*`,
)

// Info holds a single chapter's metadata and content after splitting.
type Info struct {
	Title     string // Chapter title (e.g. "第一章 陨落")
	Content   string // Full chapter text in UTF-8
	CharCount int    // Character count (runes)
}

// Split partitions full text into chapters based on Pattern.
// Returns a single "正文" chapter if no pattern matches are found.
// Pre-first-chapter content with substantial text (>10 chars) is merged into chapter 1.
func Split(content string) []Info {
	runeContent := []rune(content)
	lines := strings.Split(content, "\n")

	type matchPoint struct {
		lineIdx int
		title   string
		pos     int // byte position in the original string
	}

	var matches []matchPoint
	bytePos := 0
	for i, line := range lines {
		if Pattern.MatchString(line) {
			matches = append(matches, matchPoint{
				lineIdx: i,
				title:   strings.TrimSpace(line),
				pos:     bytePos,
			})
		}
		bytePos += len(line) + 1 // +1 for newline
	}

	// No chapter markers: return entire content as one chapter
	if len(matches) == 0 {
		return []Info{{
			Title:     "正文",
			Content:   content,
			CharCount: len(runeContent),
		}}
	}

	// Split content at match boundaries
	var chapters []Info
	for i, mp := range matches {
		title := mp.title

		start := mp.pos
		var end int
		if i+1 < len(matches) {
			end = matches[i+1].pos
		} else {
			end = len(content)
		}

		chapterContent := content[start:end]
		chapterRunes := []rune(chapterContent)

		chapters = append(chapters, Info{
			Title:     title,
			Content:   chapterContent,
			CharCount: len(chapterRunes),
		})
	}

	// Merge content before the first chapter title (preface/prologue) into chapter 1
	if matches[0].pos > 0 {
		preContent := content[:matches[0].pos]
		preText := strings.TrimSpace(preContent)
		if len([]rune(preText)) > 10 {
			chapters[0].Content = preContent + chapters[0].Content
			chapters[0].CharCount = len([]rune(chapters[0].Content))
		}
	}

	return chapters
}
