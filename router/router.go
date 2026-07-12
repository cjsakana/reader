package router

import (
	"reader/config"
	"reader/handlers"
	"reader/middleware"
	"strings"

	"github.com/gin-gonic/gin"
)

// Setup creates the gin engine, registers middleware and routes, and returns it.
func Setup(
	bookHandler *handlers.BookHandler,
	knowledgeHandler *handlers.KnowledgeHandler,
	chatHandler *handlers.ChatHandler,
	cfg *config.Config,
) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(middleware.CORS())

	// API routes
	api := r.Group("/api")
	{
		// 书籍与章节
		api.POST("/books", bookHandler.UploadBook)
		api.GET("/books", bookHandler.ListBooks)
		api.GET("/books/:id", bookHandler.GetBook)
		api.GET("/books/:id/chapters/:cid", bookHandler.GetChapter)
		api.DELETE("/books/:id", bookHandler.DeleteBook)
		api.PUT("/books/:id/progress", bookHandler.UpdateProgress)

		// 知识查询（按阅读进度过滤）
		api.GET("/books/:id/characters", knowledgeHandler.GetCharacters)
		api.GET("/books/:id/relations", knowledgeHandler.GetRelations)
		api.GET("/books/:id/events", knowledgeHandler.GetEvents)
		api.GET("/books/:id/event-characters", knowledgeHandler.GetEventCharacters)

		// 问答与对话
		api.POST("/books/:id/ask", chatHandler.Ask)
		api.GET("/books/:id/conversations", chatHandler.ListConversations)
		api.POST("/books/:id/conversations", chatHandler.CreateConversation)
		api.GET("/conversations/:id/messages", chatHandler.GetMessages)
		api.DELETE("/conversations/:id", chatHandler.DeleteConversation)
	}

	// Production: serve frontend static files
	r.StaticFile("/", "./frontend/dist/index.html")
	r.StaticFile("/index.html", "./frontend/dist/index.html")
	r.Static("/assets", "./frontend/dist/assets")
	r.Static("/static", "./frontend/dist/static")

	// SPA fallback: non-API, non-file requests return index.html
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{"error": "接口不存在"})
			return
		}
		c.File("./frontend/dist/index.html")
	})

	return r
}
