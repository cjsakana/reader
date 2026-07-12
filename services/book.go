package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reader/config"
	"reader/models"
	"reader/pkg/chapter"
	"reader/pkg/encoding"
	"reader/repository"
	"strings"
	"time"
)

// BookService handles business logic for book operations.
type BookService struct {
	repo *repository.BookRepository
	cfg  *config.Config
}

// NewBookService creates a service with the given dependencies.
func NewBookService(repo *repository.BookRepository, cfg *config.Config) *BookService {
	return &BookService{repo: repo, cfg: cfg}
}

// ImportBook imports a TXT file: save to disk, detect encoding, split chapters, persist.
func (s *BookService) ImportBook(srcFileName string, src io.Reader, fileSize int64) (*models.Book, error) {
	ext := filepath.Ext(srcFileName)
	baseName := strings.TrimSuffix(srcFileName, ext)
	saveName := fmt.Sprintf("%s_%d%s", baseName, time.Now().UnixMilli(), ext)
	savePath := filepath.Join(s.cfg.Upload.BooksDir, saveName)

	// Read all content into memory
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}

	// Detect encoding and convert to UTF-8
	content := encoding.DetectAndConvert(data)

	// Write UTF-8 content to disk
	if err := os.WriteFile(savePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	// Parse book title from filename
	title := strings.TrimSuffix(srcFileName, ext)

	// Split chapters
	chapters := chapter.Split(content)

	// Create book record
	book := &models.Book{
		Title:         title,
		Author:        "未知",
		FileName:      srcFileName,
		FilePath:      savePath,
		FileSize:      fileSize,
		TotalChars:    len([]rune(content)),
		TotalChapters: len(chapters),
		ChapterIndex:  0,
		CharOffset:    0,
	}

	if err := s.repo.CreateBook(book); err != nil {
		os.Remove(savePath)
		return nil, fmt.Errorf("create book: %w", err)
	}

	// Batch insert chapters
	chapterRecords := make([]models.Chapter, 0, len(chapters))
	for i, ch := range chapters {
		chapterRecords = append(chapterRecords, models.Chapter{
			BookID:    book.ID,
			Index:     i,
			Title:     ch.Title,
			CharCount: ch.CharCount,
			Content:   ch.Content,
		})
	}
	if len(chapterRecords) > 0 {
		if err := s.repo.BatchCreateChapters(chapterRecords); err != nil {
			s.repo.DeleteBook(book)
			os.Remove(savePath)
			return nil, fmt.Errorf("insert chapters: %w", err)
		}
	}

	return book, nil
}

// GetAllBooks returns all books ordered by most recently updated.
func (s *BookService) GetAllBooks() ([]models.Book, error) {
	return s.repo.FindAllBooks()
}

// GetBook returns a single book by ID (without chapters).
func (s *BookService) GetBook(id uint) (*models.Book, error) {
	return s.repo.FindBookByID(id)
}

// GetBookDetail returns the book with chapter summaries (metadata only, no content).
func (s *BookService) GetBookDetail(id uint) (*models.BookDetail, error) {
	book, err := s.repo.FindBookByID(id)
	if err != nil {
		return nil, err
	}

	chapters, err := s.repo.FindChaptersByBookID(id)
	if err != nil {
		return nil, err
	}

	summary := make([]models.ChapterSummary, len(chapters))
	for i, ch := range chapters {
		summary[i] = models.ChapterSummary{
			ID:        ch.ID,
			Index:     ch.Index,
			Title:     ch.Title,
			CharCount: ch.CharCount,
		}
	}

	return &models.BookDetail{
		Book:     *book,
		Chapters: summary,
	}, nil
}

// GetChapterContent returns chapter content with adjacent chapter IDs for navigation.
func (s *BookService) GetChapterContent(id uint) (*models.ChapterContent, error) {
	ch, err := s.repo.FindChapterByID(id)
	if err != nil {
		return nil, err
	}

	result := &models.ChapterContent{
		ID:        ch.ID,
		BookID:    ch.BookID,
		Index:     ch.Index,
		Title:     ch.Title,
		CharCount: ch.CharCount,
		Content:   ch.Content,
	}

	// Look up previous and next chapters
	if prev, err := s.repo.FindAdjacentChapter(ch.BookID, ch.Index-1); err == nil {
		result.PrevID = &prev.ID
	}
	if next, err := s.repo.FindAdjacentChapter(ch.BookID, ch.Index+1); err == nil {
		result.NextID = &next.ID
	}

	return result, nil
}

// DeleteBook removes a book, its chapters, and its file on disk.
func (s *BookService) DeleteBook(id uint) error {
	book, err := s.repo.FindBookByID(id)
	if err != nil {
		return err
	}

	// Remove file from disk
	os.Remove(book.FilePath)

	// Cascade delete chapters and book
	s.repo.DeleteChaptersByBookID(id)
	s.repo.DeleteBook(book)

	return nil
}

// UpdateProgress updates the reading progress for a book.
func (s *BookService) UpdateProgress(id uint, req models.ProgressRequest) error {
	return s.repo.UpdateProgress(id, req.ChapterIndex, req.CharOffset)
}
