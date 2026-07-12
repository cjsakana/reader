package models

import (
	"time"
)

// Book 图书主表
type Book struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Title         string    `json:"title"`         // 书名（从文件名解析）
	Author        string    `json:"author"`        // 作者（默认"未知"）
	FileName      string    `json:"fileName"`      // 原始文件名
	FilePath      string    `json:"-"`             // 服务端存储路径（不暴露）
	FileSize      int64     `json:"fileSize"`      // 文件大小（字节）
	TotalChars    int       `json:"totalChars"`    // 总字符数
	TotalChapters int       `json:"totalChapters"` // 总章节数
	ChapterIndex  int       `json:"chapterIndex"`  // 当前阅读章节序号（从0开始）
	CharOffset    int       `json:"charOffset"`    // 在当前章节内的字符偏移量
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Chapter 章节表（存储分割后的每章内容）
type Chapter struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BookID    uint      `json:"bookId" gorm:"index;not null"`
	Index     int       `json:"index"`     // 章节序号（从0开始）
	Title     string    `json:"title"`     // 章节标题
	CharCount int       `json:"charCount"` // 本章字符数
	Content   string    `json:"-"`         // 章节正文（不返回给列表接口）
	Parsed    bool      `json:"parsed" gorm:"default:false"` // 是否已通过LLM解析
	CreatedAt time.Time `json:"-"`
}

// ProgressRequest 更新阅读进度的请求体
type ProgressRequest struct {
	ChapterIndex int `json:"chapterIndex"`
	CharOffset   int `json:"charOffset"`
}

// BookDetail 图书详情（含章节目录）
type BookDetail struct {
	Book     Book             `json:"book"`
	Chapters []ChapterSummary `json:"chapters"`
}

// ChapterSummary 章节摘要（不含正文）
type ChapterSummary struct {
	ID        uint   `json:"id"`
	Index     int    `json:"index"`
	Title     string `json:"title"`
	CharCount int    `json:"charCount"`
}

// ChapterContent 章节内容（含正文）
type ChapterContent struct {
	ID        uint   `json:"id"`
	BookID    uint   `json:"bookId"`
	Index     int    `json:"index"`
	Title     string `json:"title"`
	CharCount int    `json:"charCount"`
	Content   string `json:"content"`
	NextID    *uint  `json:"nextId"` // 下一章 ID，null 表示最后一章
	PrevID    *uint  `json:"prevId"` // 上一章 ID，null 表示第一章
}
