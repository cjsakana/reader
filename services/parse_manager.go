package services

import (
	"context"
	"log"
	"reader/repository"
	"sync"
)

// ParseManager 管理异步章节解析任务，防止同一本书重复解析。
type ParseManager struct {
	knowledgeSvc *KnowledgeService
	repo         *repository.KnowledgeRepository
	mu           sync.Mutex
	running      map[uint]bool // bookID -> 是否正在解析中
}

// NewParseManager 创建解析管理器。
func NewParseManager(knowledgeSvc *KnowledgeService, repo *repository.KnowledgeRepository) *ParseManager {
	return &ParseManager{
		knowledgeSvc: knowledgeSvc,
		repo:         repo,
		running:      make(map[uint]bool),
	}
}

// TriggerParse 触发异步解析。非阻塞，如果该书籍已有解析任务在运行则跳过。
// chapterIndex 是读者当前的阅读进度（从0开始）。
func (m *ParseManager) TriggerParse(bookID uint, chapterIndex int) {
	m.mu.Lock()
	if m.running[bookID] {
		m.mu.Unlock()
		log.Printf("书籍 %d 已有解析任务在运行，跳过", bookID)
		return
	}
	m.running[bookID] = true
	m.mu.Unlock()
	go m.doParse(bookID, chapterIndex)
}

// doParse 执行实际的解析工作。
func (m *ParseManager) doParse(bookID uint, chapterIndex int) {
	defer func() {
		m.mu.Lock()
		delete(m.running, bookID)
		m.mu.Unlock()
	}()

	log.Printf("开始解析书籍 %d 的章节 (直到第 %d 章)", bookID, chapterIndex+1)

	// 查找所有未解析的章节
	chapters, err := m.repo.FindUnparsedChapters(bookID, chapterIndex)
	if err != nil {
		log.Printf("查找未解析章节失败 (bookID=%d): %v", bookID, err)
		return
	}

	if len(chapters) == 0 {
		log.Printf("书籍 %d 无待解析章节", bookID)
		return
	}

	log.Printf("书籍 %d 有 %d 章待解析", bookID, len(chapters))

	for _, ch := range chapters {
		log.Printf("解析中: 书籍 %d 第 %d 章", bookID, ch.Index+1)

		ctx := context.Background()
		if err := m.knowledgeSvc.ParseChapter(ctx, bookID, ch.Index, ch.Content); err != nil {
			log.Printf("解析失败 书籍 %d 第 %d 章: %v", bookID, ch.Index+1, err)
			// 继续处理后续章节，不因单章失败而中断
			continue
		}

		// 标记为已解析
		if err := m.repo.MarkChapterParsed(ch.ID); err != nil {
			log.Printf("标记章节已解析失败 第 %d 章: %v", ch.Index+1, err)
		}
	}

	log.Printf("书籍 %d 解析完成", bookID)
}
