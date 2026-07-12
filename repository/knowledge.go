package repository

import (
	"reader/models"

	"gorm.io/gorm"
)

// KnowledgeRepository 封装所有知识表（人物、别名、关系、事件）的数据库操作。
type KnowledgeRepository struct {
	db *gorm.DB
}

// NewKnowledgeRepository 创建知识仓库。
func NewKnowledgeRepository(db *gorm.DB) *KnowledgeRepository {
	return &KnowledgeRepository{db: db}
}

// ---------------------------------------------------------------------------
// 人物 (Character)
// ---------------------------------------------------------------------------

// FindCharactersByBookID 查询某书已揭示的人物（reveal_chapter <= upToChapter）。
// 按频次降序、最后出现章节降序排列。
func (r *KnowledgeRepository) FindCharactersByBookID(bookID uint, upToChapter int) ([]models.Character, error) {
	var chars []models.Character
	err := r.db.Where("book_id = ? AND reveal_chapter <= ?", bookID, upToChapter).
		Order("frequency desc, last_appear_chapter desc").Find(&chars).Error
	return chars, err
}

// FindAllCharactersByBookID 查询某书所有人物（不过滤进度，用于合并逻辑）。
func (r *KnowledgeRepository) FindAllCharactersByBookID(bookID uint) ([]models.Character, error) {
	var chars []models.Character
	err := r.db.Where("book_id = ?", bookID).Find(&chars).Error
	return chars, err
}

// CreateCharacter 插入新人物。
func (r *KnowledgeRepository) CreateCharacter(ch *models.Character) error {
	return r.db.Create(ch).Error
}

// UpdateCharacterDescription 更新人物描述（追加）。
func (r *KnowledgeRepository) UpdateCharacterDescription(id uint, desc string) error {
	return r.db.Model(&models.Character{}).Where("id = ?", id).
		Update("description", desc).Error
}

// UpdateCharacterLastAppear 更新人物最后出现章节，并增加频次。
func (r *KnowledgeRepository) UpdateCharacterLastAppear(id uint, chapter int) error {
	return r.db.Model(&models.Character{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_appear_chapter": chapter,
			"frequency":           gorm.Expr("frequency + 1"),
		}).Error
}

// UpdateCharacterFrequency 给人物增加频次（明确被提及时调用）。
func (r *KnowledgeRepository) UpdateCharacterFrequency(id uint) error {
	return r.db.Model(&models.Character{}).Where("id = ?", id).
		Update("frequency", gorm.Expr("frequency + 1")).Error
}

// BatchCreateCharacters 批量创建人物。
func (r *KnowledgeRepository) BatchCreateCharacters(chars []models.Character) error {
	if len(chars) == 0 {
		return nil
	}
	return r.db.CreateInBatches(chars, 50).Error
}

// ---------------------------------------------------------------------------
// 别名 (Alias)
// ---------------------------------------------------------------------------

// FindAliasesByCharacterIDs 查询一批人物的所有别名。
func (r *KnowledgeRepository) FindAliasesByCharacterIDs(characterIDs []uint) ([]models.Alias, error) {
	if len(characterIDs) == 0 {
		return nil, nil
	}
	var aliases []models.Alias
	err := r.db.Where("character_id IN ?", characterIDs).Find(&aliases).Error
	return aliases, err
}

// CreateAlias 插入新别名。
func (r *KnowledgeRepository) CreateAlias(a *models.Alias) error {
	return r.db.Create(a).Error
}

// BatchCreateAliases 批量创建别名。
func (r *KnowledgeRepository) BatchCreateAliases(aliases []models.Alias) error {
	if len(aliases) == 0 {
		return nil
	}
	return r.db.CreateInBatches(aliases, 50).Error
}

// ---------------------------------------------------------------------------
// 关系 (Relation)
// ---------------------------------------------------------------------------

// FindRelationsByBookID 查询某书已揭示的关系（reveal_chapter <= upToChapter）。
// 预加载关联的人物信息。
func (r *KnowledgeRepository) FindRelationsByBookID(bookID uint, upToChapter int) ([]models.Relation, error) {
	var relations []models.Relation
	err := r.db.Where("book_id = ? AND reveal_chapter <= ?", bookID, upToChapter).
		Preload("Char1").Preload("Char2").
		Order("reveal_chapter asc").Find(&relations).Error
	return relations, err
}

// FindAllRelationsByBookID 查询某书所有关系（不过滤进度）。
func (r *KnowledgeRepository) FindAllRelationsByBookID(bookID uint) ([]models.Relation, error) {
	var relations []models.Relation
	err := r.db.Where("book_id = ?", bookID).
		Preload("Char1").Preload("Char2").Find(&relations).Error
	return relations, err
}

// CreateRelation 插入新关系。
func (r *KnowledgeRepository) CreateRelation(rel *models.Relation) error {
	return r.db.Create(rel).Error
}

// BatchCreateRelations 批量创建关系。
func (r *KnowledgeRepository) BatchCreateRelations(rels []models.Relation) error {
	if len(rels) == 0 {
		return nil
	}
	return r.db.CreateInBatches(rels, 50).Error
}

// UpdateRelationEndChapter 设置关系的结束章节。
func (r *KnowledgeRepository) UpdateRelationEndChapter(id uint, endChapter uint) error {
	return r.db.Model(&models.Relation{}).Where("id = ?", id).
		Update("end_chapter", endChapter).Error
}

// ---------------------------------------------------------------------------
// 事件 (Event)
// ---------------------------------------------------------------------------

// FindEventsByBookID 查询某书已揭示的事件（reveal_chapter <= upToChapter）。
func (r *KnowledgeRepository) FindEventsByBookID(bookID uint, upToChapter int) ([]models.Event, error) {
	var events []models.Event
	err := r.db.Where("book_id = ? AND reveal_chapter <= ?", bookID, upToChapter).
		Order("chapter_number asc").Find(&events).Error
	return events, err
}

// CreateEvent 插入新事件。
func (r *KnowledgeRepository) CreateEvent(e *models.Event) error {
	return r.db.Create(e).Error
}

// BatchCreateEvents 批量创建事件。
func (r *KnowledgeRepository) BatchCreateEvents(events []models.Event) error {
	if len(events) == 0 {
		return nil
	}
	return r.db.CreateInBatches(events, 50).Error
}

// ---------------------------------------------------------------------------
// 事件-人物关联 (EventCharacter)
// ---------------------------------------------------------------------------

// FindEventCharactersByEventIDs 查询一批事件关联的人物。
func (r *KnowledgeRepository) FindEventCharactersByEventIDs(eventIDs []uint) ([]models.EventCharacter, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}
	var ecs []models.EventCharacter
	err := r.db.Where("event_id IN ?", eventIDs).Preload("Character").Find(&ecs).Error
	return ecs, err
}

// BatchCreateEventCharacters 批量创建事件-人物关联。
func (r *KnowledgeRepository) BatchCreateEventCharacters(ecs []models.EventCharacter) error {
	if len(ecs) == 0 {
		return nil
	}
	return r.db.CreateInBatches(ecs, 50).Error
}

// ---------------------------------------------------------------------------
// 章节解析标记
// ---------------------------------------------------------------------------

// MarkChapterParsed 标记某章已解析。
func (r *KnowledgeRepository) MarkChapterParsed(chapterID uint) error {
	return r.db.Model(&models.Chapter{}).Where("id = ?", chapterID).
		Update("parsed", true).Error
}

// FindUnparsedChapters 查询某书所有 index <= upToIndex 且 parsed = false 的章节，按序号升序排列。
func (r *KnowledgeRepository) FindUnparsedChapters(bookID uint, upToIndex int) ([]models.Chapter, error) {
	var chapters []models.Chapter
	err := r.db.Where("book_id = ? AND \"index\" <= ? AND parsed = ?", bookID, upToIndex, false).
		Order("\"index\" asc").Find(&chapters).Error
	return chapters, err
}
