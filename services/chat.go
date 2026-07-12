package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"reader/llm"
	"reader/milvus"
	"reader/models"
	"reader/repository"
)

// AskResponse is the result returned by the Ask method.
type AskResponse struct {
	Answer     string        `json:"answer"`
	References []AskRef      `json:"references"`
}

// AskRef is a single reference citation.
type AskRef struct {
	Chapter int    `json:"chapter"`
	Text    string `json:"text"`
}

// ChatService handles RAG question answering and conversation management.
type ChatService struct {
	repo          *repository.ChatRepository
	knowledgeRepo *repository.KnowledgeRepository
	llmClient     *llm.Client
	milvusClient  *milvus.Client
}

// NewChatService creates a chat service.
func NewChatService(
	repo *repository.ChatRepository,
	knowledgeRepo *repository.KnowledgeRepository,
	llmClient *llm.Client,
	milvusClient *milvus.Client,
) *ChatService {
	return &ChatService{
		repo:          repo,
		knowledgeRepo: knowledgeRepo,
		llmClient:     llmClient,
		milvusClient:  milvusClient,
	}
}

// Ask performs the RAG QA pipeline: embed question → search Milvus → build context → LLM answer.
func (s *ChatService) Ask(ctx context.Context, bookID uint, conversationID uint, question string, currentChapter int) (*AskResponse, error) {
	// 1. 向量检索
	var refs []AskRef
	if s.milvusClient != nil {
		var err error
		refs, err = s.searchMilvus(ctx, bookID, currentChapter, question)
		if err != nil {
			log.Printf("Milvus 搜索失败: %v（将继续使用 SQLite 知识）", err)
		}
	}

	// 2. 获取已知知识摘要
	knownChars, _ := s.knowledgeRepo.FindCharactersByBookID(bookID, currentChapter)
	knownRels, _ := s.knowledgeRepo.FindRelationsByBookID(bookID, currentChapter)
	knownEvents, _ := s.knowledgeRepo.FindEventsByBookID(bookID, currentChapter)

	// 3. 获取对话历史（最近6条消息）
	history := ""
	if conversationID > 0 {
		msgs, err := s.repo.FindMessagesByConversationID(conversationID)
		if err == nil && len(msgs) > 0 {
			var sb strings.Builder
			sb.WriteString("【对话历史】\n")
			start := 0
			if len(msgs) > 6 {
				start = len(msgs) - 6
			}
			for _, m := range msgs[start:] {
				sb.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
			}
			history = sb.String()
		}
	}

	// 4. 组装上下文
	var ctxBuilder strings.Builder
	ctxBuilder.WriteString(history)

	if len(refs) > 0 {
		ctxBuilder.WriteString("\n【检索到的相关文本片段】\n")
		for i, ref := range refs {
			ctxBuilder.WriteString(fmt.Sprintf("[%d] (第%d章) %s\n", i+1, ref.Chapter+1, ref.Text))
		}
	}

	ctxBuilder.WriteString("\n【已知人物】\n")
	for _, ch := range knownChars {
		ctxBuilder.WriteString(fmt.Sprintf("- %s：%s\n", ch.Name, ch.Description))
	}

	ctxBuilder.WriteString("\n【已知关系】\n")
	for _, r := range knownRels {
		char1Name, char2Name := "", ""
		if r.Char1 != nil {
			char1Name = r.Char1.Name
		}
		if r.Char2 != nil {
			char2Name = r.Char2.Name
		}
		ctxBuilder.WriteString(fmt.Sprintf("- %s 与 %s：%s（%s）\n", char1Name, char2Name, r.RelationType, r.Description))
	}

	ctxBuilder.WriteString("\n【已知事件】\n")
	for _, e := range knownEvents {
		ctxBuilder.WriteString(fmt.Sprintf("- 第%d章：%s（重要性 %d/5）\n", e.ChapterNumber+1, e.Summary, e.Importance))
	}

	// 5. 调用 LLM
	systemPrompt := `你是小说阅读助手，帮助读者理解已读内容。严格规则：
1. 只能基于提供的【检索文本片段】、【已知人物】、【已知关系】、【已知事件】和【对话历史】回答。
2. 不得使用预训练的外部知识，不得推测未读内容。
3. 如果信息不足以回答问题，明确告知"在你当前的阅读进度中尚未提供相关信息"。
4. 回答时标注不确定性（如"根据目前读到的内容，似乎……"）。
5. 引用来源时使用 [N] 标记对应检索片段的编号。
6. 回答使用中文。`

	userPrompt := fmt.Sprintf("【读者当前阅读到第 %d 章】\n\n%s\n\n【读者提问】：%s", currentChapter+1, ctxBuilder.String(), question)

	answer, err := s.llmClient.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM 问答失败: %w", err)
	}

	// 6. 保存消息
	if conversationID > 0 {
		now := time.Now()
		s.repo.CreateMessage(&models.Message{
			ConversationID: conversationID,
			Role:           "user",
			Content:        question,
			CreatedAt:      now,
		})
		s.repo.CreateMessage(&models.Message{
			ConversationID: conversationID,
			Role:           "assistant",
			Content:        answer,
			CreatedAt:      now.Add(time.Second),
		})

		// 第一个问题后自动生成标题
		count, _ := s.repo.CountMessagesByConversationID(conversationID)
		if count <= 2 {
			title, err := s.generateTitle(ctx, question)
			if err == nil && title != "" {
				_ = s.repo.UpdateConversationTitle(conversationID, title)
			}
		}
	}

	return &AskResponse{
		Answer:     answer,
		References: refs,
	}, nil
}

// searchMilvus 执行向量搜索并返回引文列表。
func (s *ChatService) searchMilvus(ctx context.Context, bookID uint, currentChapter int, question string) ([]AskRef, error) {
	vectors, err := s.llmClient.Embed(ctx, []string{question})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("empty embedding result")
	}

	results, err := s.milvusClient.Search(ctx, vectors[0], int64(bookID), int64(currentChapter), 10)
	if err != nil {
		return nil, err
	}

	// Limit snippet length.
	refs := make([]AskRef, 0, len(results))
	for _, r := range results {
		text := r.Content
		if len([]rune(text)) > 150 {
			text = string([]rune(text)[:150]) + "……"
		}
		refs = append(refs, AskRef{
			Chapter: int(r.ChapterNum),
			Text:    text,
		})
	}
	return refs, nil
}

// generateTitle 根据第一个问题生成对话标题（15字以内）。
func (s *ChatService) generateTitle(ctx context.Context, question string) (string, error) {
	sysPrompt := `你是一个标题生成器。根据用户的问题生成一个简洁的对话标题（15字以内）。
只输出标题文本，不要包含任何其他内容。`

	return s.llmClient.Chat(ctx, sysPrompt, question)
}

// CreateConversation 创建新对话。
func (s *ChatService) CreateConversation(bookID uint, title string) (*models.Conversation, error) {
	if title == "" {
		title = "新对话"
	}
	conv := &models.Conversation{
		BookID:    bookID,
		Title:     title,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateConversation(conv); err != nil {
		return nil, err
	}
	return conv, nil
}

// GetConversations 获取某书的所有对话。
func (s *ChatService) GetConversations(bookID uint) ([]models.Conversation, error) {
	return s.repo.FindConversationsByBookID(bookID)
}

// GetMessages 获取对话的消息历史。
func (s *ChatService) GetMessages(conversationID uint) ([]models.Message, error) {
	return s.repo.FindMessagesByConversationID(conversationID)
}

// DeleteConversation 删除对话及其所有消息。
func (s *ChatService) DeleteConversation(id uint) error {
	return s.repo.DeleteConversation(id)
}
