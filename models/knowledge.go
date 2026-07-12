package models

import "time"

// Character 人物表
type Character struct {
	ID                uint   `json:"id" gorm:"primaryKey"`
	BookID            uint   `json:"bookId" gorm:"index;not null"`
	Name              string `json:"name"`                                                // 标准名称
	Description       string `json:"description"`                                         // 简介/摘要（持续更新）
	RevealChapter     int    `json:"revealChapter" gorm:"not null"`                       // 首次出现的章节序号（从0开始）
	LastAppearChapter int    `json:"lastAppearChapter" gorm:"default:0"`                  // 最后出现的章节序号
	Frequency         int    `json:"frequency" gorm:"default:1"`                          // 出现频次（每解析到一次+1）
}

// Alias 人物别名表
type Alias struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	CharacterID   uint   `json:"characterId" gorm:"index;not null"`
	Alias         string `json:"alias" gorm:"not null"`             // 别名
	RevealChapter int    `json:"revealChapter" gorm:"not null"`      // 别名首次出现的章节序号
}

// Relation 人物关系表
type Relation struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	BookID        uint   `json:"bookId" gorm:"index;not null"`
	Char1ID       uint   `json:"char1Id" gorm:"not null"`           // 人物1 ID
	Char2ID       uint   `json:"char2Id" gorm:"not null"`           // 人物2 ID
	RelationType  string `json:"relationType"`                      // 关系类型：朋友、敌对、情侣、家人等
	Description   string `json:"description"`                       // 关系简述
	RevealChapter int    `json:"revealChapter" gorm:"not null"`      // 关系明确出现的章节
	StartChapter  *uint  `json:"startChapter"`                      // 关系开始章节
	EndChapter    *uint  `json:"endChapter"`                        // 关系结束章节（NULL表示持续）

	// 外键关联（用于预加载）
	Char1 *Character `json:"char1,omitempty" gorm:"foreignKey:Char1ID"`
	Char2 *Character `json:"char2,omitempty" gorm:"foreignKey:Char2ID"`
}

// Event 事件表
type Event struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	BookID        uint   `json:"bookId" gorm:"index;not null"`
	ChapterNumber int    `json:"chapterNumber" gorm:"not null"`     // 发生在哪一章（从0开始）
	Summary       string `json:"summary" gorm:"not null"`           // 事件简述
	Importance    int    `json:"importance" gorm:"default:0"`       // 重要程度评分（1-5）
	RevealChapter int    `json:"revealChapter" gorm:"not null"`      // 事件真相揭示的章节
}

// EventCharacter 事件-人物关联表
type EventCharacter struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	EventID     uint   `json:"eventId" gorm:"index;not null"`
	CharacterID uint   `json:"characterId" gorm:"not null"`
	Role        string `json:"role"`                                // 参与者、见证者、提及

	// 外键关联
	Character *Character `json:"character,omitempty" gorm:"foreignKey:CharacterID"`
}

// Conversation 对话会话表
type Conversation struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BookID    uint      `json:"bookId" gorm:"index;not null"`
	Title     string    `json:"title"`                              // 对话标题
	CreatedAt time.Time `json:"createdAt"`
}

// Message 对话消息表
type Message struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	ConversationID uint      `json:"conversationId" gorm:"index;not null"`
	Role           string    `json:"role" gorm:"not null"`          // "user" 或 "assistant"
	Content        string    `json:"content" gorm:"not null"`
	CreatedAt      time.Time `json:"createdAt"`
}
