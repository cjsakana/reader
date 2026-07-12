package handlers

import (
	"reader/models"
	"reader/pkg/response"
	"reader/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ChatHandler 对话和问答的 HTTP 处理器。
type ChatHandler struct {
	chatSvc *services.ChatService
	bookSvc *services.BookService
}

// NewChatHandler 创建对话处理器。
func NewChatHandler(chatSvc *services.ChatService, bookSvc *services.BookService) *ChatHandler {
	return &ChatHandler{chatSvc: chatSvc, bookSvc: bookSvc}
}

// Ask 处理问答请求。
// POST /api/books/:id/ask
func (h *ChatHandler) Ask(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParam)
		return
	}

	var req struct {
		ConversationID uint   `json:"conversation_id"`
		Question       string `json:"question"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadRequest)
		return
	}
	if req.Question == "" {
		response.Error(c, response.CodeBadRequest)
		return
	}

	book, err := h.bookSvc.GetBook(uint(bookID))
	if err != nil {
		response.Error(c, response.CodeNotFound)
		return
	}

	result, err := h.chatSvc.Ask(c.Request.Context(), uint(bookID), req.ConversationID, req.Question, book.ChapterIndex)
	if err != nil {
		response.ErrorMsg(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// ListConversations 获取某书的所有对话列表。
// GET /api/books/:id/conversations
func (h *ChatHandler) ListConversations(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParam)
		return
	}

	convs, err := h.chatSvc.GetConversations(uint(bookID))
	if err != nil {
		response.ErrorMsg(c, response.CodeInternalError, err.Error())
		return
	}
	if convs == nil {
		convs = []models.Conversation{}
	}
	response.Success(c, convs)
}

// CreateConversation 创建新对话。
// POST /api/books/:id/conversations
func (h *ChatHandler) CreateConversation(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParam)
		return
	}

	var req struct {
		Title string `json:"title"`
	}
	_ = c.ShouldBindJSON(&req)

	conv, err := h.chatSvc.CreateConversation(uint(bookID), req.Title)
	if err != nil {
		response.ErrorMsg(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, conv)
}

// GetMessages 获取对话的消息历史。
// GET /api/conversations/:id/messages
func (h *ChatHandler) GetMessages(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParam)
		return
	}

	msgs, err := h.chatSvc.GetMessages(uint(convID))
	if err != nil {
		response.ErrorMsg(c, response.CodeInternalError, err.Error())
		return
	}
	if msgs == nil {
		msgs = []models.Message{}
	}
	response.Success(c, msgs)
}

// DeleteConversation 删除对话。
// DELETE /api/conversations/:id
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParam)
		return
	}

	if err := h.chatSvc.DeleteConversation(uint(id)); err != nil {
		response.Error(c, response.CodeNotFound)
		return
	}

	response.SuccessMsg(c, "删除成功")
}
