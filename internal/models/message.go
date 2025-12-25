package models

import (
	"time"

	"gorm.io/gorm"
)

// Message represents a message configuration
type Message struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	PayloadSize int       `json:"payload_size"` // Payload size (bytes)
	MessageType string    `json:"message_type"` // Message type
	Retained    bool      `json:"retained"`     // Retained flag
	Duplicate   bool      `json:"duplicate"`    // Duplicate flag
	QoSLevel    int       `json:"qos_level"`    // QoS level
	Status      string    `json:"status"`       // Status
	StartTime   time.Time `json:"start_time"`   // Start time
	EndTime     time.Time `json:"end_time"`     // End time
	CreatedAt   time.Time `json:"created_at"`   // Creation time
}

// TableName specifies the table name for Message
func (Message) TableName() string {
	return "messages"
}

// MessageModel defines the interface for operating message data
type MessageModel struct {
	DB *gorm.DB
}

// MessageGorm represents the GORM operations for Message
// GetAll retrieves all message records
func (m *MessageModel) GetAll() ([]*Message, error) {
	var messages []*Message
	result := m.DB.Order("created_at DESC").Find(&messages)
	return messages, result.Error
}

// GetByID retrieves a message record by ID
func (m *MessageModel) GetByID(id int64) (*Message, error) {
	var message Message
	result := m.DB.First(&message, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &message, nil
}

// Insert inserts a new message record
func (m *MessageModel) Insert(message *Message) error {
	message.CreatedAt = time.Now()
	result := m.DB.Create(message)
	return result.Error
}

// Update updates a message record
func (m *MessageModel) Update(message *Message) error {
	result := m.DB.Save(message)
	return result.Error
}

// Delete deletes a message record
func (m *MessageModel) Delete(id int64) error {
	result := m.DB.Delete(&Message{}, id)
	return result.Error
}

// MessageGorm provides GORM-based database operations for Message
type MessageGorm struct{}

// GetAll retrieves all message records using GORM
func (g *MessageGorm) GetAll(db *gorm.DB) ([]*Message, error) {
	var messages []*Message
	result := db.Order("created_at DESC").Find(&messages)
	return messages, result.Error
}

// GetByID retrieves a message record by ID using GORM
func (g *MessageGorm) GetByID(db *gorm.DB, id int64) (*Message, error) {
	var message Message
	result := db.First(&message, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &message, nil
}

// Insert inserts a new message record using GORM
func (g *MessageGorm) Insert(db *gorm.DB, message *Message) error {
	message.CreatedAt = time.Now()
	result := db.Create(message)
	return result.Error
}

// Update updates a message record using GORM
func (g *MessageGorm) Update(db *gorm.DB, message *Message) error {
	result := db.Save(message)
	return result.Error
}

// Delete deletes a message record using GORM
func (g *MessageGorm) Delete(db *gorm.DB, id int64) error {
	result := db.Delete(&Message{}, id)
	return result.Error
}
