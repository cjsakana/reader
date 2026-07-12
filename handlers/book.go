package handlers

import (
	"reader/models"
	"reader/pkg/response"
	"reader/services"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// BookHandler holds all book-related HTTP handlers.
type BookHandler struct {
	svc          *services.BookService
	parseManager *services.ParseManager
}

// NewBookHandler creates a handler with the given service and parse manager.
func NewBookHandler(svc *services.BookService, parseManager *services.ParseManager) *BookHandler {
	return &BookHandler{svc: svc, parseManager: parseManager}
}

// UploadBook handles POST /api/books — upload a TXT file, split chapters, and persist.
func (h *BookHandler) UploadBook(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, response.CodeBadRequest)
		return
	}

	// Validate .txt extension
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".txt") {
		response.Error(c, response.CodeFileTypeErr)
		return
	}

	src, err := file.Open()
	if err != nil {
		response.Error(c, response.CodeFileIOError)
		return
	}
	defer src.Close()

	book, err := h.svc.ImportBook(file.Filename, src, file.Size)
	if err != nil {
		response.ErrorMsg(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, book)
}

// ListBooks handles GET /api/books — returns all books.
func (h *BookHandler) ListBooks(c *gin.Context) {
	books, err := h.svc.GetAllBooks()
	if err != nil {
		response.Error(c, response.CodeDBError)
		return
	}
	if books == nil {
		books = []models.Book{}
	}
	response.Success(c, books)
}

// GetBook handles GET /api/books/:id — returns book detail with chapter summaries.
func (h *BookHandler) GetBook(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParam)
		return
	}

	detail, err := h.svc.GetBookDetail(uint(id))
	if err != nil {
		response.Error(c, response.CodeNotFound)
		return
	}

	response.Success(c, detail)
}

// GetChapter handles GET /api/books/:id/chapters/:cid — returns chapter content.
func (h *BookHandler) GetChapter(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("cid"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParam)
		return
	}

	content, err := h.svc.GetChapterContent(uint(cid))
	if err != nil {
		response.Error(c, response.CodeNotFound)
		return
	}

	response.Success(c, content)
}

// DeleteBook handles DELETE /api/books/:id — removes a book and its chapters.
func (h *BookHandler) DeleteBook(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParam)
		return
	}

	if err := h.svc.DeleteBook(uint(id)); err != nil {
		response.Error(c, response.CodeNotFound)
		return
	}

	response.SuccessMsg(c, "删除成功")
}

// UpdateProgress handles PUT /api/books/:id/progress — saves reading progress.
func (h *BookHandler) UpdateProgress(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParam)
		return
	}

	var req models.ProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadRequest)
		return
	}

	if err := h.svc.UpdateProgress(uint(id), req); err != nil {
		response.Error(c, response.CodeNotFound)
		return
	}

	// 触发异步章节解析（防剧透：只解析已读到的章节）
	if h.parseManager != nil {
		go h.parseManager.TriggerParse(uint(id), req.ChapterIndex)
	}

	response.SuccessMsg(c, "进度已保存")
}
