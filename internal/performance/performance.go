package performance

import (
	"mqttbench/internal/db"
	"mqttbench/internal/models"
)

// Service 性能测试服务
type Service struct {
	performanceGorm *models.PerformanceGorm
}

// NewService 创建新的性能测试服务实例
func NewService() *Service {
	return &Service{
		performanceGorm: &models.PerformanceGorm{},
	}
}

// GetPerformanceTests 获取所有性能测试记录
func (s *Service) GetPerformanceTests() ([]*models.Performance, error) {
	return s.performanceGorm.GetAll(db.DB)
}
