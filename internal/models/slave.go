package models

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// Slave represents a slave configuration
type Slave struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name"`                         // Slave name
	MqttHost    string    `json:"mqtt_host"`                    // MQTT Host address
	MqttPort    int       `json:"mqtt_port"`                    // MQTT Port number
	SlaveHost   string    `json:"slave_host"`                   // Slave Host address
	SlavePort   int       `json:"slave_port"`                   // Slave Port number
	ClientID    string    `json:"client_id"`                    // Client ID
	KeepAlive   int       `json:"keep_alive" gorm:"default:60"` // Keep alive interval
	Topic       string    `json:"topic"`                        // MQTT Topic
	QoS         int       `json:"qos" gorm:"column:qos"`        // MQTT QoS, default is 0
	Start       int       `json:"start"`                        // Start value
	Step        int       `json:"step"`                         // Step value (替代原来的End字段)
	AckTopic    string    `json:"ack_topic"`                    // ACK Topic
	Status      string    `json:"status"`                       // Slave status (online/offline)
	Connections int       `json:"connections"`                  // Number of MQTT connections
	CreatedAt   time.Time `json:"created_at"`                   // Creation time (slave first registered time)
	UpdatedAt   time.Time `json:"updated_at"`                   // Update time
}

// TableName specifies the table name for Slave
func (Slave) TableName() string {
	return "slaves"
}

// SlaveModel defines the interface for operating slave data
type SlaveModel struct {
	DB *gorm.DB
}

// GetAll retrieves all slave records
func (m *SlaveModel) GetAll() ([]*Slave, error) {
	var slaves []*Slave
	result := m.DB.Find(&slaves)
	if result.Error != nil {
		return nil, result.Error
	}

	// 添加调试日志
	log.Printf("Database query returned %d slaves", len(slaves))
	for _, slave := range slaves {
		log.Printf("Slave in database: ID=%d, Name=%s, Status=%s, Connections=%d, CreatedAt=%v, UpdatedAt=%v, MqttHost=%s, MqttPort=%d, SlaveHost=%s, SlavePort=%d",
			slave.ID, slave.Name, slave.Status, slave.Connections, slave.CreatedAt, slave.UpdatedAt, slave.MqttHost, slave.MqttPort, slave.SlaveHost, slave.SlavePort)
	}

	// 修复可能存在的无效时间戳
	for _, slave := range slaves {
		// 检查 CreatedAt 是否为零值或无效值
		if slave.CreatedAt.IsZero() || slave.CreatedAt.Year() <= 1 {
			slave.CreatedAt = time.Now()
			// 更新数据库中的记录
			saveResult := m.DB.Save(slave)
			if saveResult.Error != nil {
				// 记录错误但不中断操作
				// 可以考虑添加日志记录
				log.Printf("Error fixing CreatedAt for slave %d: %v", slave.ID, saveResult.Error)
			} else {
				log.Printf("Fixed CreatedAt for slave %d", slave.ID)
			}
		}
		// 如果 UpdatedAt 也是无效值，设置为当前时间
		if slave.UpdatedAt.IsZero() || slave.UpdatedAt.Year() <= 1 {
			slave.UpdatedAt = time.Now()
			// 更新数据库中的记录
			saveResult := m.DB.Save(slave)
			if saveResult.Error != nil {
				// 记录错误但不中断操作
				// 可以考虑添加日志记录
				log.Printf("Error fixing UpdatedAt for slave %d: %v", slave.ID, saveResult.Error)
			} else {
				log.Printf("Fixed UpdatedAt for slave %d", slave.ID)
			}
		}
	}

	return slaves, result.Error
}

// GetByID retrieves a slave record by ID
func (m *SlaveModel) GetByID(id int64) (*Slave, error) {
	var slave Slave
	result := m.DB.First(&slave, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	// 修复可能存在的无效时间戳
	if slave.CreatedAt.IsZero() || slave.CreatedAt.Year() <= 1 {
		slave.CreatedAt = time.Now()
		saveResult := m.DB.Save(&slave)
		if saveResult.Error != nil {
			// 记录错误但不中断操作
		}
	}
	if slave.UpdatedAt.IsZero() || slave.UpdatedAt.Year() <= 1 {
		slave.UpdatedAt = time.Now()
		saveResult := m.DB.Save(&slave)
		if saveResult.Error != nil {
			// 记录错误但不中断操作
		}
	}

	return &slave, nil
}

// Insert inserts a new slave record
func (m *SlaveModel) Insert(slave *Slave) error {
	// 设置创建和更新时间
	now := time.Now()
	slave.CreatedAt = now
	slave.UpdatedAt = now

	// 如果状态未设置，则默认为offline
	if slave.Status == "" {
		slave.Status = "offline"
	}

	// 如果连接数未设置，则默认为0
	if slave.Connections == 0 {
		slave.Connections = 0
	}

	result := m.DB.Create(slave)
	return result.Error
}

// UpdateWithConnections 更新slave记录，包含连接数
func (m *SlaveModel) UpdateWithConnections(slave *Slave) error {
	// 只更新更新时间，保持创建时间不变
	slave.UpdatedAt = time.Now()

	// 不更新创建时间，保持为第一次注册的时间
	result := m.DB.Model(slave).Select("name", "mqtt_host", "mqtt_port", "slave_host", "slave_port", "client_id", "keep_alive", "topic", "qos", "start", "step", "status", "connections", "updated_at").Updates(slave)

	// 添加详细的错误日志
	if result.Error != nil {
		log.Printf("Failed to update slave %d with connections: %v", slave.ID, result.Error)
		log.Printf("Slave data: %+v", slave)
	} else {
		log.Printf("Successfully updated slave %d with connections, rows affected: %d", slave.ID, result.RowsAffected)
	}

	return result.Error
}

// UpdateWithoutConnections 更新slave记录，不包含连接数
func (m *SlaveModel) UpdateWithoutConnections(slave *Slave) error {
	// 只更新更新时间，保持创建时间不变
	slave.UpdatedAt = time.Now()

	// 不更新创建时间，保持为第一次注册的时间，不更新connections字段
	result := m.DB.Model(slave).Select("name", "mqtt_host", "mqtt_port", "slave_host", "slave_port", "client_id", "keep_alive", "topic", "qos", "start", "step", "status", "updated_at").Updates(slave)

	// 添加详细的错误日志
	if result.Error != nil {
		log.Printf("Failed to update slave %d without connections: %v", slave.ID, result.Error)
		log.Printf("Slave data: %+v", slave)
	} else {
		log.Printf("Successfully updated slave %d without connections, rows affected: %d", slave.ID, result.RowsAffected)
	}

	return result.Error
}

// Delete deletes a slave record
func (m *SlaveModel) Delete(id int64) error {
	result := m.DB.Delete(&Slave{}, id)
	return result.Error
}

// SlaveGorm provides GORM-based database operations for Slave
type SlaveGorm struct{}

// GetAll retrieves all slave records using GORM
func (g *SlaveGorm) GetAll(db *gorm.DB) ([]*Slave, error) {
	var slaves []*Slave
	result := db.Find(&slaves)
	return slaves, result.Error
}

// GetByID retrieves a slave record by ID using GORM
func (g *SlaveGorm) GetByID(db *gorm.DB, id int64) (*Slave, error) {
	var slave Slave
	result := db.First(&slave, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &slave, nil
}

// Insert inserts a new slave record using GORM
func (g *SlaveGorm) Insert(db *gorm.DB, slave *Slave) error {
	now := time.Now()
	slave.CreatedAt = now
	slave.UpdatedAt = now

	// 如果状态未设置，则默认为offline
	if slave.Status == "" {
		slave.Status = "offline"
	}

	// 如果连接数未设置，则默认为0
	if slave.Connections == 0 {
		slave.Connections = 0
	}

	result := db.Create(slave)
	return result.Error
}

// UpdateWithConnections updates a slave record using GORM with connections field
func (g *SlaveGorm) UpdateWithConnections(db *gorm.DB, slave *Slave) error {
	// 只更新更新时间，保持创建时间不变
	slave.UpdatedAt = time.Now()

	// 不更新创建时间，保持为第一次注册的时间
	result := db.Model(slave).Select("name", "mqtt_host", "mqtt_port", "slave_host", "slave_port", "client_id", "keep_alive", "topic", "qos", "start", "step", "status", "connections", "updated_at").Updates(slave)
	return result.Error
}

// UpdateWithoutConnections updates a slave record using GORM without connections field
func (g *SlaveGorm) UpdateWithoutConnections(db *gorm.DB, slave *Slave) error {
	// 只更新更新时间，保持创建时间不变
	slave.UpdatedAt = time.Now()

	// 不更新创建时间，保持为第一次注册的时间，不更新connections字段
	result := db.Model(slave).Select("name", "mqtt_host", "mqtt_port", "slave_host", "slave_port", "client_id", "keep_alive", "topic", "qos", "start", "step", "status", "updated_at").Updates(slave)
	return result.Error
}

// Delete deletes a slave record using GORM
func (g *SlaveGorm) Delete(db *gorm.DB, id int64) error {
	result := db.Delete(&Slave{}, id)
	return result.Error
}
