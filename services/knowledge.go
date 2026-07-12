package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"reader/config"
	"reader/llm"
	"reader/milvus"
	"reader/pkg/chunker"
	"reader/repository"
)

// KnowledgeService 处理章节解析、知识抽取与融合的业务逻辑。
type KnowledgeService struct {
	repo         *repository.KnowledgeRepository
	llmClient    *llm.Client
	milvusClient *milvus.Client
	cfg          *config.Config
}

// NewKnowledgeService 创建知识服务。
func NewKnowledgeService(
	repo *repository.KnowledgeRepository,
	llmClient *llm.Client,
	milvusClient *milvus.Client,
	cfg *config.Config,
) *KnowledgeService {
	return &KnowledgeService{
		repo:         repo,
		llmClient:    llmClient,
		milvusClient: milvusClient,
		cfg:          cfg,
	}
}

// ---------------------------------------------------------------------------
// 章节解析流水线
// ---------------------------------------------------------------------------

// ParseChapter 对单个章节执行完整的解析流水线：
// 1. 文本分块
// 2. 生成嵌入向量并存入 Milvus
// 3. 实体识别（人物+别名）
// 4. 关系抽取
// 5. 事件抽取
// 6. 知识融合入库
func (s *KnowledgeService) ParseChapter(ctx context.Context, bookID uint, chapterIndex int, chapterContent string) error {
	log.Printf("开始解析第 %d 章 (bookID=%d)", chapterIndex+1, bookID)

	// 1. 文本分块
	chunkCfg := chunker.DefaultConfig()
	chunks := chunker.Split(chapterContent, chunkCfg)
	if len(chunks) == 0 {
		log.Printf("第 %d 章无有效文本块，跳过", chapterIndex+1)
		return nil
	}
	log.Printf("第 %d 章分为 %d 个文本块", chapterIndex+1, len(chunks))

	// 2. 生成嵌入 + 存入 Milvus
	_ = s.storeChunks(ctx, bookID, chapterIndex, chunks)

	// 3-5. 顺序调用 LLM 抽取知识
	knownSummary := s.buildKnownSummary(bookID)

	entities, err := s.extractEntities(ctx, chapterContent, knownSummary)
	if err != nil {
		log.Printf("警告: 实体识别失败: %v", err)
		// 继续处理而非直接返回错误
	}
	log.Printf("第 %d 章识别到 %d 个人物", chapterIndex+1, len(entities))

	relations, err := s.extractRelations(ctx, chapterContent, knownSummary, entities)
	if err != nil {
		log.Printf("警告: 关系抽取失败: %v", err)
	}
	log.Printf("第 %d 章识别到 %d 个关系", chapterIndex+1, len(relations))

	events, err := s.extractEvents(ctx, chapterContent)
	if err != nil {
		log.Printf("警告: 事件抽取失败: %v", err)
	}
	log.Printf("第 %d 章识别到 %d 个事件", chapterIndex+1, len(events))

	// 6. 知识融合
	if err := s.mergeKnowledge(bookID, chapterIndex, entities, relations, events); err != nil {
		return fmt.Errorf("merge knowledge: %w", err)
	}

	log.Printf("第 %d 章解析完成", chapterIndex+1)
	return nil
}

// storeChunks 生成嵌入向量并存入 Milvus。
func (s *KnowledgeService) storeChunks(ctx context.Context, bookID uint, chapterIndex int, chunks []chunker.Chunk) error {
	if s.milvusClient == nil {
		return nil
	}

	texts := make([]string, len(chunks))
	for i, ch := range chunks {
		texts[i] = ch.Text
	}

	vectors, err := s.llmClient.Embed(ctx, texts)
	if err != nil {
		log.Printf("警告: 嵌入生成失败: %v", err)
		return err
	}

	docs := make([]milvus.ChunkDoc, len(chunks))
	for i, ch := range chunks {
		docs[i] = milvus.ChunkDoc{
			ID:       fmt.Sprintf("b%d_c%d_ch%d", bookID, chapterIndex, i),
			Content:  ch.Text,
			Vector:   vectors[i],
			MetaData: map[string]any{
				"book_id":       int64(bookID),
				"chapter_number": int64(chapterIndex),
				"chunk_index":    int64(i),
			},
		}
	}

	return s.milvusClient.StoreChunks(ctx, docs)
}

// buildKnownSummary 构建已知知识摘要，供 LLM 提示词使用。
func (s *KnowledgeService) buildKnownSummary(bookID uint) string {
	// 获取已有人物（不限进度，因为解析时需要全量知识做去重）
	chars, err := s.repo.FindAllCharactersByBookID(bookID)
	if err != nil || len(chars) == 0 {
		return "暂无已知人物。"
	}

	var sb strings.Builder
	sb.WriteString("【已知人物列表】\n")
	for _, ch := range chars {
		sb.WriteString(fmt.Sprintf("- %s：%s\n", ch.Name, ch.Description))
	}

	// 获取已有别名
	var charIDs []uint
	for i := range chars {
		charIDs = append(charIDs, chars[i].ID)
	}
	if len(charIDs) > 0 {
		aliases, err := s.repo.FindAliasesByCharacterIDs(charIDs)
		if err == nil && len(aliases) > 0 {
			sb.WriteString("\n【已知别名映射】\n")
			for _, a := range aliases {
				// 找到对应人物名
				for i := range chars {
					if chars[i].ID == a.CharacterID {
						sb.WriteString(fmt.Sprintf("- %s → %s\n", a.Alias, chars[i].Name))
						break
					}
				}
			}
		}
	}

	return sb.String()
}

// ---------------------------------------------------------------------------
// LLM 抽取方法
// ---------------------------------------------------------------------------

const entityPrompt = `你是小说阅读助手，将严格基于【本章文本】和【已知知识摘要】提取信息。
规则：
1. 只提取本章明确出现或说明的信息。
2. 不确定的关系/身份不要标注，宁可模糊。
3. 别名需对应到已知人物，若无法确定则暂列为新人物。
4. 输出纯 JSON 格式，不要包含任何 Markdown 标记或其他文字。

【已知知识摘要】：
%s

【本章文本】：
%s

请输出：
{
  "characters": [
    {"name": "角色名", "aliases": ["别名1", "别名2"], "description": "本章中的角色简介"}
  ]
}`

func (s *KnowledgeService) extractEntities(ctx context.Context, chapterContent, knownSummary string) ([]ExtractedEntity, error) {
	prompt := fmt.Sprintf(entityPrompt, knownSummary, chapterContent)
	resp, err := s.llmClient.Chat(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, err
	}

	var result struct {
		Characters []ExtractedEntity `json:"characters"`
	}
	if err := extractJSON(resp, &result); err != nil {
		// 重试一次，明确要求 JSON
		retryPrompt := prompt + "\n\n注意：请只输出 JSON，不要包含 Markdown 标记。"
		resp, err2 := s.llmClient.Chat(ctx, systemPrompt, retryPrompt)
		if err2 != nil {
			return nil, fmt.Errorf("retry entity extraction: %w (original: %w)", err2, err)
		}
		if err2 := extractJSON(resp, &result); err2 != nil {
			return nil, fmt.Errorf("extract JSON after retry: %w", err2)
		}
	}

	return result.Characters, nil
}

const relationPrompt = `你是小说阅读助手，请基于【本章文本】提取人物之间的明确关系。
规则：
1. 只提取本章明确展示的关系，不要推断。
2. 关系类型可以是：朋友、敌对、情侣、家人、师徒、同事、主仆、盟友、其他。
3. 输出纯 JSON 格式，不要包含任何 Markdown 标记或其他文字。

【已知人物与关系】：
%s

【本章文本】：
%s

请输出：
{
  "relations": [
    {"char1": "人物A", "char2": "人物B", "type": "关系类型", "description": "关系简述"}
  ]
}`

func (s *KnowledgeService) extractRelations(ctx context.Context, chapterContent, knownSummary string, entities []ExtractedEntity) ([]ExtractedRelation, error) {
	// 将新识别的人物也加入已知摘要
	if len(entities) > 0 {
		var sb strings.Builder
		sb.WriteString(knownSummary)
		sb.WriteString("\n【本章新识别人物】\n")
		for _, e := range entities {
			sb.WriteString(fmt.Sprintf("- %s：%s\n", e.Name, e.Description))
		}
		knownSummary = sb.String()
	}

	prompt := fmt.Sprintf(relationPrompt, knownSummary, chapterContent)
	resp, err := s.llmClient.Chat(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, err
	}

	var result struct {
		Relations []ExtractedRelation `json:"relations"`
	}
	if err := extractJSON(resp, &result); err != nil {
		return nil, fmt.Errorf("extract relations JSON: %w", err)
	}

	return result.Relations, nil
}

const eventPrompt = `你是小说阅读助手，请从【本章文本】中提取关键事件。
规则：
1. 只提取本章明确发生的事件。
2. 重要性评分标准（1-5分）：
   - 5分：改变故事走向的关键转折
   - 4分：显著推动主线发展
   - 3分：重要的支线或角色成长
   - 2分：包含有用信息的日常事件
   - 1分：纯日常描写，信息量低
3. 输出纯 JSON 格式，不要包含任何 Markdown 标记或其他文字。

【本章文本】：
%s

请输出：
{
  "events": [
    {"summary": "事件简述", "involved_characters": ["角色A", "角色B"], "importance": 3}
  ]
}`

func (s *KnowledgeService) extractEvents(ctx context.Context, chapterContent string) ([]ExtractedEvent, error) {
	prompt := fmt.Sprintf(eventPrompt, chapterContent)
	resp, err := s.llmClient.Chat(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, err
	}

	var result struct {
		Events []ExtractedEvent `json:"events"`
	}
	if err := extractJSON(resp, &result); err != nil {
		return nil, fmt.Errorf("extract events JSON: %w", err)
	}

	return result.Events, nil
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

const systemPrompt = `你是小说阅读助手，将严格基于提供的文本提取信息。
规则：
1. 只提取文本中明确出现或说明的信息。
2. 不确定的信息不要标注，宁可模糊。
3. 不要推断未来情节，不添加外部知识。
4. 始终输出纯 JSON 格式。`

// extractJSON 从 LLM 响应中提取 JSON，自动剥离 Markdown 代码块标记。
func extractJSON(raw string, target interface{}) error {
	raw = strings.TrimSpace(raw)

	// 去除 Markdown 代码块标记
	if strings.HasPrefix(raw, "```json") {
		raw = strings.TrimPrefix(raw, "```json")
	} else if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
	}
	if strings.HasSuffix(raw, "```") {
		raw = strings.TrimSuffix(raw, "```")
	}
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return fmt.Errorf("empty response")
	}

	return json.Unmarshal([]byte(raw), target)
}

// GetKnowledgeSummary 获取书籍的已揭示知识摘要（用于前端展示和 RAG 上下文）。
func (s *KnowledgeService) GetKnowledgeSummary(ctx context.Context, bookID uint, upToChapter int) (map[string]interface{}, error) {
	chars, err := s.repo.FindCharactersByBookID(bookID, upToChapter)
	if err != nil {
		return nil, err
	}

	// 查询别名
	var charIDs []uint
	for i := range chars {
		charIDs = append(charIDs, chars[i].ID)
	}
	aliases, _ := s.repo.FindAliasesByCharacterIDs(charIDs)
	aliasMap := make(map[uint][]string)
	for _, a := range aliases {
		if a.RevealChapter <= upToChapter {
			aliasMap[a.CharacterID] = append(aliasMap[a.CharacterID], a.Alias)
		}
	}

	relations, err := s.repo.FindRelationsByBookID(bookID, upToChapter)
	if err != nil {
		relations = nil
	}

	events, err := s.repo.FindEventsByBookID(bookID, upToChapter)
	if err != nil {
		events = nil
	}

	// 查询事件-人物关联
	var eventIDs []uint
	for i := range events {
		eventIDs = append(eventIDs, events[i].ID)
	}
	eventChars, _ := s.repo.FindEventCharactersByEventIDs(eventIDs)

	return map[string]interface{}{
		"characters":    chars,
		"aliases":       aliasMap,
		"relations":     relations,
		"events":        events,
		"eventChars": eventChars,
	}, nil
}

// GetCharacters 获取已揭示的人物及别名列表（用于 API 响应）。
func (s *KnowledgeService) GetCharacters(bookID uint, upToChapter int) ([]interface{}, error) {
	chars, err := s.repo.FindCharactersByBookID(bookID, upToChapter)
	if err != nil {
		return nil, err
	}

	var charIDs []uint
	for i := range chars {
		charIDs = append(charIDs, chars[i].ID)
	}
	aliases, _ := s.repo.FindAliasesByCharacterIDs(charIDs)

	aliasMap := make(map[uint][]string)
	for _, a := range aliases {
		if a.RevealChapter <= upToChapter {
			aliasMap[a.CharacterID] = append(aliasMap[a.CharacterID], a.Alias)
		}
	}

	result := make([]interface{}, 0, len(chars))
	for _, ch := range chars {
		als := aliasMap[ch.ID]
		if als == nil {
			als = []string{}
		}
		result = append(result, map[string]interface{}{
			"id":                ch.ID,
			"bookId":            ch.BookID,
			"name":              ch.Name,
			"description":       ch.Description,
			"revealChapter":     ch.RevealChapter,
			"lastAppearChapter": ch.LastAppearChapter,
			"frequency":         ch.Frequency,
			"aliases":           als,
		})
	}
	return result, nil
}

// GetRelations 获取已揭示的关系列表。
func (s *KnowledgeService) GetRelations(bookID uint, upToChapter int) ([]interface{}, error) {
	rels, err := s.repo.FindRelationsByBookID(bookID, upToChapter)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, 0, len(rels))
	for _, r := range rels {
		item := map[string]interface{}{
			"id":            r.ID,
			"bookId":        r.BookID,
			"relationType":  r.RelationType,
			"description":   r.Description,
			"revealChapter": r.RevealChapter,
			"startChapter":  r.StartChapter,
			"endChapter":    r.EndChapter,
		}
		if r.Char1 != nil {
			item["char1"] = map[string]interface{}{
				"id":   r.Char1.ID,
				"name": r.Char1.Name,
			}
		}
		if r.Char2 != nil {
			item["char2"] = map[string]interface{}{
				"id":   r.Char2.ID,
				"name": r.Char2.Name,
			}
		}
		result = append(result, item)
	}
	return result, nil
}

// GetEvents 获取已揭示的事件列表。
func (s *KnowledgeService) GetEvents(bookID uint, upToChapter int) ([]interface{}, error) {
	events, err := s.repo.FindEventsByBookID(bookID, upToChapter)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, 0, len(events))
	for _, e := range events {
		result = append(result, map[string]interface{}{
			"id":            e.ID,
			"bookId":        e.BookID,
			"chapterNumber": e.ChapterNumber,
			"summary":       e.Summary,
			"importance":    e.Importance,
			"revealChapter": e.RevealChapter,
		})
	}
	return result, nil
}

// GetEventCharacters 获取事件-人物关联列表。
func (s *KnowledgeService) GetEventCharacters(bookID uint) ([]interface{}, error) {
	// 获取该书所有事件
	chars, _ := s.repo.FindAllCharactersByBookID(bookID)
	_ = chars
	return []interface{}{}, nil
}
