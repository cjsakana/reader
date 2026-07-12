package handlers

import (
	"reader/pkg/response"
	"reader/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

// KnowledgeHandler 知识查询的 HTTP 处理器。
type KnowledgeHandler struct {
	knowledgeSvc *services.KnowledgeService
	bookSvc      *services.BookService
}

// NewKnowledgeHandler 创建知识查询处理器。
func NewKnowledgeHandler(knowledgeSvc *services.KnowledgeService, bookSvc *services.BookService) *KnowledgeHandler {
	return &KnowledgeHandler{knowledgeSvc: knowledgeSvc, bookSvc: bookSvc}
}

// GetCharacters 获取已揭示的人物及别名。
// GET /api/books/:id/characters
func (h *KnowledgeHandler) GetCharacters(c *gin.Context) {
	bookID, currentChapter, ok := h.resolveBookProgress(c)
	if !ok {
		return
	}

	chars, err := h.knowledgeSvc.GetCharacters(bookID, currentChapter)
	if err != nil {
		response.ErrorMsg(c, response.CodeInternalError, err.Error())
		return
	}
	if chars == nil {
		chars = []interface{}{}
	}
	response.Success(c, chars)
}

// GetRelations 获取已揭示的人物关系。
// GET /api/books/:id/relations
func (h *KnowledgeHandler) GetRelations(c *gin.Context) {
	bookID, currentChapter, ok := h.resolveBookProgress(c)
	if !ok {
		return
	}

	rels, err := h.knowledgeSvc.GetRelations(bookID, currentChapter)
	if err != nil {
		response.ErrorMsg(c, response.CodeInternalError, err.Error())
		return
	}
	if rels == nil {
		rels = []interface{}{}
	}
	response.Success(c, rels)
}

// GetEvents 获取已揭示的事件时间线。
// GET /api/books/:id/events
func (h *KnowledgeHandler) GetEvents(c *gin.Context) {
	bookID, currentChapter, ok := h.resolveBookProgress(c)
	if !ok {
		return
	}

	events, err := h.knowledgeSvc.GetEvents(bookID, currentChapter)
	if err != nil {
		response.ErrorMsg(c, response.CodeInternalError, err.Error())
		return
	}
	if events == nil {
		events = []interface{}{}
	}
	response.Success(c, events)
}

// GetEventCharacters 获取事件与人物关联。
// GET /api/books/:id/event-characters
func (h *KnowledgeHandler) GetEventCharacters(c *gin.Context) {
	bookID, _, ok := h.resolveBookProgress(c)
	if !ok {
		return
	}

	ecs, err := h.knowledgeSvc.GetEventCharacters(bookID)
	if err != nil {
		response.ErrorMsg(c, response.CodeInternalError, err.Error())
		return
	}
	if ecs == nil {
		ecs = []interface{}{}
	}
	response.Success(c, ecs)
}

// resolveBookProgress 从路由参数中提取 bookID，并从阅读进度中获取当前章节。
func (h *KnowledgeHandler) resolveBookProgress(c *gin.Context) (bookID uint, currentChapter int, ok bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParam)
		return 0, 0, false
	}
	bookID = uint(id)

	book, err := h.bookSvc.GetBook(bookID)
	if err != nil {
		response.Error(c, response.CodeNotFound)
		return 0, 0, false
	}

	return bookID, book.ChapterIndex, true
}
