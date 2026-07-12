package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"reader/config"
	"reader/handlers"
	"reader/llm"
	"reader/milvus"
	"reader/models"
	"reader/repository"
	"reader/router"
	"reader/services"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	path := flag.String("f", "config.yaml", "path to config file")
	flag.Parse()
	cfg := config.Load(*path)

	// Initialize database
	db, err := gorm.Open(sqlite.Open(cfg.Database.DBPath), &gorm.Config{})
	if err != nil {
		panic("数据库初始化失败: " + err.Error())
	}
	if err := db.AutoMigrate(&models.Book{}, &models.Chapter{},
		&models.Character{}, &models.Alias{},
		&models.Relation{}, &models.Event{}, &models.EventCharacter{},
		&models.Conversation{}, &models.Message{}); err != nil {
		panic("数据库迁移失败: " + err.Error())
	}
	fmt.Println("数据库初始化完成")

	// Initialize LLM client
	ctx := context.Background()
	llmClient, err := llm.NewClient(ctx, cfg.LLM)
	if err != nil {
		log.Printf("警告: LLM 客户端初始化失败: %v (章节解析和问答功能将不可用)", err)
	}
	fmt.Println("LLM初始化完成")

	// Initialize Milvus client
	var milvusClient *milvus.Client
	milvusClient, err = milvus.New(ctx, cfg.Milvus.Address,
		cfg.Milvus.Username, cfg.Milvus.Password, cfg.Milvus.DBName,
		cfg.Milvus.Collection, cfg.Milvus.Dimension)
	if err != nil {
		log.Printf("警告: Milvus 客户端初始化失败: %v (向量检索功能将不可用)", err)
	} else {
		fmt.Println("Milvus初始化完成")
	}

	// Dependency injection
	repo := repository.NewBookRepository(db)
	svc := services.NewBookService(repo, cfg)

	// Knowledge pipeline
	knowledgeRepo := repository.NewKnowledgeRepository(db)
	var knowledgeSvc *services.KnowledgeService
	var parseManager *services.ParseManager
	if llmClient != nil {
		knowledgeSvc = services.NewKnowledgeService(knowledgeRepo, llmClient, milvusClient, cfg)
		parseManager = services.NewParseManager(knowledgeSvc, knowledgeRepo)
	}

	// Knowledge handler
	var knowledgeHandler *handlers.KnowledgeHandler
	var chatHandler *handlers.ChatHandler
	if knowledgeSvc != nil {
		knowledgeHandler = handlers.NewKnowledgeHandler(knowledgeSvc, svc)

		// Chat service and handler
		if llmClient != nil {
			chatRepo := repository.NewChatRepository(db)
			chatSvc := services.NewChatService(chatRepo, knowledgeRepo, llmClient, milvusClient)
			chatHandler = handlers.NewChatHandler(chatSvc, svc)
		}
	}

	handler := handlers.NewBookHandler(svc, parseManager)

	// Setup routes and start server
	r := router.Setup(handler, knowledgeHandler, chatHandler, cfg)

	fmt.Println("服务启动在 http://localhost:" + cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		panic("启动服务失败: " + err.Error())
	}
}
