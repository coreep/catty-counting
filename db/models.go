package db

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	TelegramID       int64 `gorm:"uniqueIndex"`
	TelegramUserName string
	Chats            []Chat
	Messages         []Message
}

type Chat struct {
	gorm.Model
	UserID     uint   `gorm:"index"`
	Summary    string `gorm:"type:text"`
	TelegramID int64
	User       User
	Messages   []Message
}

type Message struct {
	gorm.Model
	UserID      uint   `gorm:"index"`
	ChatID      uint   `gorm:"index"`
	Text        string `gorm:"type:text"`
	TelegramIDs []int
	User        User
	Chat        Chat
	Files       []File
}

type File struct {
	gorm.Model
	MessageID    uint `gorm:"index"`
	Size         int64
	MimeType     string `gorm:"type:varchar(256)"`
	OriginalName string `gorm:"type:varchar(1024)"`
	Summary      string `gorm:"type:text"`
	TelegramID   string
	Message      Message
	ExposedFile  *ExposedFile
}

type ExposedFile struct {
	gorm.Model
	FileID uint   `gorm:"index"`
	Key    string `gorm:"type:varchar(256);uniqueIndex"`
	File   File
}
