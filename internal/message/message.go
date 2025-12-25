package message

import (
	"mqttbench/internal/db"
	"mqttbench/internal/models"
)

// Service 消息测试服务
type Service struct {
	messageGorm *models.MessageGorm
}

// NewService 创建新的消息测试服务实例
func NewService() *Service {
	return &Service{
		messageGorm: &models.MessageGorm{},
	}
}

// GetMessageTests 获取所有消息测试记录
func (s *Service) GetMessageTests() ([]*models.Message, error) {
	return s.messageGorm.GetAll(db.DB)
}
