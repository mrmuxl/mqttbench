package models

import (
	"time"

	"gorm.io/gorm"
)

// Performance represents a performance configuration
type Performance struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TestDuration int       `json:"test_duration"` // Test duration (seconds)
	MessageRate  int       `json:"message_rate"`  // Message rate (messages per second)
	MessageSize  int       `json:"message_size"`  // Message size (bytes)
	QoSLevel     int       `json:"qos_level"`     // QoS level
	Status       string    `json:"status"`        // Status
	StartTime    time.Time `json:"start_time"`    // Start time
	EndTime      time.Time `json:"end_time"`      // End time
	CreatedAt    time.Time `json:"created_at"`    // Creation time
}

// TableName specifies the table name for Performance
func (Performance) TableName() string {
	return "performances"
}

// PerformanceGorm represents the GORM operations for Performance
type PerformanceGorm struct{}

// GetAll retrieves all performance records
func (g *PerformanceGorm) GetAll(db *gorm.DB) ([]*Performance, error) {
	var performances []*Performance
	result := db.Order("created_at DESC").Find(&performances)
	return performances, result.Error
}

// GetByID retrieves a performance record by ID
func (g *PerformanceGorm) GetByID(db *gorm.DB, id int64) (*Performance, error) {
	var performance Performance
	result := db.First(&performance, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &performance, nil
}

// Insert inserts a new performance record
func (g *PerformanceGorm) Insert(db *gorm.DB, performance *Performance) error {
	performance.CreatedAt = time.Now()
	result := db.Create(performance)
	return result.Error
}

// Update updates a performance record
func (g *PerformanceGorm) Update(db *gorm.DB, performance *Performance) error {
	result := db.Save(performance)
	return result.Error
}

// Delete deletes a performance record
func (g *PerformanceGorm) Delete(db *gorm.DB, id int64) error {
	result := db.Delete(&Performance{}, id)
	return result.Error
}

// TestFunction is a test function to verify package import
func TestFunction() string {
	return "Test function in models package"
}
