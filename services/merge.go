package services

import (
	"context"
	"fmt"
	"strings"

	"reader/models"
)

// ExtractedEntity 单个人物抽取结果
type ExtractedEntity struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases"`
	Description string   `json:"description"`
}

// ExtractedRelation 单个关系抽取结果
type ExtractedRelation struct {
	Char1       string `json:"char1"`
	Char2       string `json:"char2"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ExtractedEvent 单个事件抽取结果
type ExtractedEvent struct {
	Summary            string   `json:"summary"`
	InvolvedCharacters []string `json:"involved_characters"`
	Importance         int      `json:"importance"`
}

// mergeKnowledge 将本章新抽取的知识与已知知识融合，存入数据库。
func (s *KnowledgeService) mergeKnowledge(
	bookID uint,
	chapterIndex int,
	entities []ExtractedEntity,
	relations []ExtractedRelation,
	events []ExtractedEvent,
) error {
	// 获取已有人物
	existingChars, err := s.repo.FindAllCharactersByBookID(bookID)
	if err != nil {
		return fmt.Errorf("query existing characters: %w", err)
	}

	// 构建名称→人物映射
	nameToChar := make(map[string]*models.Character)
	var charIDs []uint
	for i := range existingChars {
		nameToChar[existingChars[i].Name] = &existingChars[i]
		charIDs = append(charIDs, existingChars[i].ID)
	}

	// 查询已有别名，构建别名→人物映射
	aliasToChar := make(map[string]*models.Character)
	if len(charIDs) > 0 {
		aliases, err := s.repo.FindAliasesByCharacterIDs(charIDs)
		if err != nil {
			return fmt.Errorf("query existing aliases: %w", err)
		}
		for _, a := range aliases {
			for i := range existingChars {
				if a.CharacterID == existingChars[i].ID {
					aliasToChar[a.Alias] = &existingChars[i]
					break
				}
			}
		}
	}

	// ---- 处理人物 ----
	newChars := make([]models.Character, 0)
	newAliases := make([]models.Alias, 0)

	for _, ent := range entities {
		if strings.TrimSpace(ent.Name) == "" {
			continue
		}
		name := strings.TrimSpace(ent.Name)

		// 1. 尝试通过标准名匹配
		if char, ok := nameToChar[name]; ok {
			appendDescription(s, char, ent.Description)
			addNewAliases(&newAliases, &aliasToChar, char, ent.Aliases, name, chapterIndex)
			s.repo.UpdateCharacterLastAppear(char.ID, chapterIndex)
			continue
		}

		// 2. 尝试通过别名匹配
		if char, ok := aliasToChar[name]; ok {
			appendDescription(s, char, ent.Description)
			addNewAliases(&newAliases, &aliasToChar, char, ent.Aliases, name, chapterIndex)
			s.repo.UpdateCharacterLastAppear(char.ID, chapterIndex)
			continue
		}

		// 3. 新人物
		newChars = append(newChars, models.Character{
			BookID:            bookID,
			Name:              name,
			Description:       ent.Description,
			RevealChapter:     chapterIndex,
			LastAppearChapter: chapterIndex,
			Frequency:         1,
		})
	}

	// 批量创建新人物
	if len(newChars) > 0 {
		if err := s.repo.BatchCreateCharacters(newChars); err != nil {
			return fmt.Errorf("batch create characters: %w", err)
		}
		// 更新映射以便后续关系处理
		allChars, _ := s.repo.FindAllCharactersByBookID(bookID)
		for i := range allChars {
			if _, ok := nameToChar[allChars[i].Name]; !ok {
				nameToChar[allChars[i].Name] = &allChars[i]
			}
		}
	}

	// 批量创建新别名
	if len(newAliases) > 0 {
		if err := s.repo.BatchCreateAliases(newAliases); err != nil {
			return fmt.Errorf("batch create aliases: %w", err)
		}
	}

	// ---- 处理关系 ----
	existingRels, err := s.repo.FindAllRelationsByBookID(bookID)
	if err != nil {
		return fmt.Errorf("query existing relations: %w", err)
	}

	newRels := make([]models.Relation, 0)
	for _, rel := range relations {
		char1Name := strings.TrimSpace(rel.Char1)
		char2Name := strings.TrimSpace(rel.Char2)
		if char1Name == "" || char2Name == "" || char1Name == char2Name {
			continue
		}

		char1 := findCharByName(nameToChar, aliasToChar, char1Name)
		char2 := findCharByName(nameToChar, aliasToChar, char2Name)
		if char1 == nil || char2 == nil {
			continue
		}

		dup := false
		for i := range existingRels {
			er := &existingRels[i]
			if isSamePair(er, char1.ID, char2.ID) && er.RelationType == rel.Type {
				dup = true
				break
			}
			if isSamePair(er, char1.ID, char2.ID) && er.RelationType != rel.Type && er.EndChapter == nil {
				endCh := uint(chapterIndex)
				_ = s.repo.UpdateRelationEndChapter(er.ID, endCh)
			}
		}

		if !dup {
			startCh := uint(chapterIndex)
			newRels = append(newRels, models.Relation{
				BookID:        bookID,
				Char1ID:       char1.ID,
				Char2ID:       char2.ID,
				RelationType:  rel.Type,
				Description:   rel.Description,
				RevealChapter: chapterIndex,
				StartChapter:  &startCh,
			})
		}
	}
	if len(newRels) > 0 {
		if err := s.repo.BatchCreateRelations(newRels); err != nil {
			return fmt.Errorf("batch create relations: %w", err)
		}
	}

	// ---- 处理事件 ----
	newEventChars := make([]models.EventCharacter, 0)

	for _, evt := range events {
		if strings.TrimSpace(evt.Summary) == "" {
			continue
		}
		importance := evt.Importance
		if importance < 1 {
			importance = 1
		}
		if importance > 5 {
			importance = 5
		}

		event := models.Event{
			BookID:        bookID,
			ChapterNumber: chapterIndex,
			Summary:       strings.TrimSpace(evt.Summary),
			Importance:    importance,
			RevealChapter: chapterIndex,
		}
		if err := s.repo.CreateEvent(&event); err != nil {
			return fmt.Errorf("create event: %w", err)
		}

		for _, charName := range evt.InvolvedCharacters {
			charName = strings.TrimSpace(charName)
			if charName == "" {
				continue
			}
			char := findCharByName(nameToChar, aliasToChar, charName)
			if char != nil {
				newEventChars = append(newEventChars, models.EventCharacter{
					EventID:     event.ID,
					CharacterID: char.ID,
					Role:        "参与者",
				})
			}
		}
	}
	if len(newEventChars) > 0 {
		_ = s.repo.BatchCreateEventCharacters(newEventChars)
	}

	return nil
}

// appendDescription 追加人物描述。如果已有描述则合并后用 LLM 总结。
func appendDescription(s *KnowledgeService, char *models.Character, desc string) {
	if desc == "" {
		return
	}
	if char.Description == "" {
		_ = s.repo.UpdateCharacterDescription(char.ID, desc)
		return
	}
	// 已有描述 + 新描述 → LLM 总结
	ctx := context.TODO()
	merged, err := s.summarizeDescription(ctx, char.Description, desc)
	if err != nil {
		// 降级：直接拼接
		merged = char.Description + "；" + desc
	}
	_ = s.repo.UpdateCharacterDescription(char.ID, merged)
}

// summarizeDescription 用 LLM 将已有描述与新信息合并总结为简洁摘要（50字内）。
func (s *KnowledgeService) summarizeDescription(ctx context.Context, existing, newInfo string) (string, error) {
	sysPrompt := `你是一个角色简介编辑器。请将现有的角色简介和新获取的信息合并成一段简洁、通顺的总结（不超过50字），保留最重要的人物特征和关系。
规则：
1. 只输出总结文本，不要包含任何格式标记。
2. 用中文输出。
3. 不要添加不存在于新旧信息中的内容。`

	userPrompt := fmt.Sprintf("【已有简介】：%s\n【新信息】：%s\n\n请输出合并后的简介：", existing, newInfo)
	return s.llmClient.Chat(ctx, sysPrompt, userPrompt)
}

// addNewAliases 将尚未存在的别名加入待创建列表。
func addNewAliases(newAliases *[]models.Alias, aliasToChar *map[string]*models.Character, char *models.Character, aliases []string, mainName string, chapterIndex int) {
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" || alias == mainName {
			continue
		}
		if _, exists := (*aliasToChar)[alias]; !exists {
			*newAliases = append(*newAliases, models.Alias{
				CharacterID:   char.ID,
				Alias:         alias,
				RevealChapter: chapterIndex,
			})
			(*aliasToChar)[alias] = char
		}
	}
}

// findCharByName 通过标准名或别名查找人物。
func findCharByName(nameToChar map[string]*models.Character, aliasToChar map[string]*models.Character, name string) *models.Character {
	if char, ok := nameToChar[name]; ok {
		return char
	}
	if char, ok := aliasToChar[name]; ok {
		return char
	}
	return nil
}

// isSamePair 检查关系是否涉及同一对人物（忽略顺序）。
func isSamePair(r *models.Relation, id1, id2 uint) bool {
	return (r.Char1ID == id1 && r.Char2ID == id2) || (r.Char1ID == id2 && r.Char2ID == id1)
}
