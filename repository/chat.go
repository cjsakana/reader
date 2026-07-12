package repository

import (
	"reader/models"

	"gorm.io/gorm"
)

// ChatRepository 封装对话和消息的数据库操作。
type ChatRepository struct {
	db *gorm.DB
}

// NewChatRepository 创建对话仓库。
func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

// CreateConversation 创建新对话。
func (r *ChatRepository) CreateConversation(conv *models.Conversation) error {
	return r.db.Create(conv).Error
}

// UpdateConversationTitle 更新对话标题。
func (r *ChatRepository) UpdateConversationTitle(id uint, title string) error {
	return r.db.Model(&models.Conversation{}).Where("id = ?", id).Update("title", title).Error
}

// FindConversationsByBookID 查询某书的所有对话，按创建时间降序。
func (r *ChatRepository) FindConversationsByBookID(bookID uint) ([]models.Conversation, error) {
	var convs []models.Conversation
	err := r.db.Where("book_id = ?", bookID).
		Order("created_at desc").Find(&convs).Error
	return convs, err
}

// FindConversationByID 通过 ID 查询对话。
func (r *ChatRepository) FindConversationByID(id uint) (*models.Conversation, error) {
	var conv models.Conversation
	err := r.db.First(&conv, id).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// DeleteConversation 删除对话及其所有消息。
func (r *ChatRepository) DeleteConversation(id uint) error {
	r.db.Where("conversation_id = ?", id).Delete(&models.Message{})
	return r.db.Delete(&models.Conversation{}, id).Error
}

// CreateMessage 保存一条消息。
func (r *ChatRepository) CreateMessage(msg *models.Message) error {
	return r.db.Create(msg).Error
}

// FindMessagesByConversationID 查询某对话的所有消息，按时间升序。
func (r *ChatRepository) FindMessagesByConversationID(convID uint) ([]models.Message, error) {
	var msgs []models.Message
	err := r.db.Where("conversation_id = ?", convID).
		Order("created_at asc").Find(&msgs).Error
	return msgs, err
}

// CountMessagesByConversationID 统计对话的消息数。
func (r *ChatRepository) CountMessagesByConversationID(convID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Message{}).Where("conversation_id = ?", convID).Count(&count).Error
	return count, err
}
