package db

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type MessageDirection string

const (
	MessageDirectionFromUser    MessageDirection = "from-user"
	MessageDirectionToUser      MessageDirection = "to-user"
	MessageDirectionSystemToLlm MessageDirection = "system-to-llm"
	MessageDirectionLlmToSystem MessageDirection = "llm-to-system"
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
	User       *User
	Messages   []Message
}

type Message struct {
	gorm.Model
	UserID      uint             `gorm:"index"`
	ChatID      uint             `gorm:"index"`
	Text        string           `gorm:"type:text"`
	TelegramIDs []int            `gorm:"serializer:json"`
	Direction   MessageDirection `gorm:"type:varchar(16)"`
	User        *User
	Chat        *Chat
	Files       []File
}

// A file provided by the User
type File struct {
	gorm.Model
	MessageID    uint   `gorm:"index"`
	BlobKey      string `gorm:"varchar(128)"`
	Size         int64
	MimeType     string `gorm:"type:varchar(256)"`
	OriginalName string `gorm:"type:varchar(1024)"`
	Summary      string `gorm:"type:text"`
	TelegramID   string
	Message      *Message
	ExposedFile  *ExposedFile
	Receipts     []Receipt
}

// Reperesents an access link for LLM to download the file
type ExposedFile struct {
	gorm.Model
	FileID uint   `gorm:"index"`
	Key    string `gorm:"type:varchar(256);uniqueIndex"`
	File   File
}

// Represents a parsed receipt/document extracted from a File
type Receipt struct {
	gorm.Model
	FileID         uint            `gorm:"index"`
	TotalBeforeTax decimal.Decimal `gorm:"type:decimal(20,2)" json:"total_before_tax"`
	Tax            decimal.Decimal `gorm:"type:decimal(20,2)" json:"tax"`
	TotalWithTax   decimal.Decimal `gorm:"type:decimal(20,2)" json:"total_with_tax"`
	Origin         string          `gorm:"type:text" json:"origin"`
	Recipient      string          `gorm:"type:text" json:"recipient"`
	Details        string          `gorm:"type:text" json:"details"`
	Summary        string          `gorm:"type:text" json:"summary"`
	File           *File
	Products       []Product
}

// Product represents an item parsed from a Receipt
type Product struct {
	gorm.Model
	ReceiptID      uint            `gorm:"index"`
	Title          string          `gorm:"type:text" json:"title"`
	Details        string          `gorm:"type:text" json:"details"`
	TotalBeforeTax decimal.Decimal `gorm:"type:decimal(20,2)" json:"total_before_tax"`
	Tax            decimal.Decimal `gorm:"type:decimal(20,2)" json:"tax"`
	TotalWithTax   decimal.Decimal `gorm:"type:decimal(20,2)" json:"total_with_tax"`
	Receipt        *Receipt
	Categories     []Category `gorm:"many2many:product_categories" json:"categories"`
}

// Category groups products
type Category struct {
	gorm.Model
	Title    string    `gorm:"type:text" json:"title"`
	Details  string    `gorm:"type:text" json:"details"`
	Products []Product `gorm:"many2many:product_categories"`
}

// ProductCategory is the join table for many-to-many Product<->Category
type ProductCategory struct {
	gorm.Model
	ProductID  uint `gorm:"index"`
	CategoryID uint `gorm:"index"`
}
