package repository

import (
	"reader/models"
	"time"

	"gorm.io/gorm"
)

// BookRepository encapsulates all database operations for books and chapters.
type BookRepository struct {
	db *gorm.DB
}

// NewBookRepository creates a repository with the given DB connection.
func NewBookRepository(db *gorm.DB) *BookRepository {
	return &BookRepository{db: db}
}

// DB returns the underlying *gorm.DB (used by main.go for AutoMigrate).
func (r *BookRepository) DB() *gorm.DB {
	return r.db
}

// CreateBook inserts a book record.
func (r *BookRepository) CreateBook(book *models.Book) error {
	return r.db.Create(book).Error
}

// BatchCreateChapters inserts chapters in batches of 100.
func (r *BookRepository) BatchCreateChapters(chapters []models.Chapter) error {
	return r.db.CreateInBatches(chapters, 100).Error
}

// FindAllBooks returns all books ordered by updated_at descending.
func (r *BookRepository) FindAllBooks() ([]models.Book, error) {
	var books []models.Book
	err := r.db.Order("updated_at desc").Find(&books).Error
	return books, err
}

// FindBookByID returns a single book by primary key.
func (r *BookRepository) FindBookByID(id uint) (*models.Book, error) {
	var book models.Book
	err := r.db.First(&book, id).Error
	if err != nil {
		return nil, err
	}
	return &book, nil
}

// FindChaptersByBookID returns all chapters for a book, ordered by index ascending.
func (r *BookRepository) FindChaptersByBookID(bookID uint) ([]models.Chapter, error) {
	var chapters []models.Chapter
	err := r.db.Where("book_id = ?", bookID).Order("\"index\" asc").Find(&chapters).Error
	return chapters, err
}

// FindChapterByID returns a single chapter by primary key.
func (r *BookRepository) FindChapterByID(id uint) (*models.Chapter, error) {
	var ch models.Chapter
	err := r.db.First(&ch, id).Error
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// FindAdjacentChapter finds the chapter at the given bookID and index.
// Returns nil, gorm.ErrRecordNotFound if no such chapter exists (boundary).
func (r *BookRepository) FindAdjacentChapter(bookID uint, index int) (*models.Chapter, error) {
	var ch models.Chapter
	err := r.db.Where("book_id = ? AND \"index\" = ?", bookID, index).First(&ch).Error
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// DeleteChaptersByBookID deletes all chapters belonging to a book.
func (r *BookRepository) DeleteChaptersByBookID(bookID uint) error {
	return r.db.Where("book_id = ?", bookID).Delete(&models.Chapter{}).Error
}

// DeleteBook deletes a book record.
func (r *BookRepository) DeleteBook(book *models.Book) error {
	return r.db.Delete(book).Error
}

// UpdateProgress updates the reading progress fields on a book.
func (r *BookRepository) UpdateProgress(id uint, chapterIndex int, charOffset int) error {
	return r.db.Model(&models.Book{}).Where("id = ?", id).Updates(map[string]interface{}{
		"chapter_index": chapterIndex,
		"char_offset":   charOffset,
		"updated_at":    time.Now(),
	}).Error
}
