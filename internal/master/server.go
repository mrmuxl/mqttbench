package master

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"mqttbench/internal/db"
	"mqttbench/internal/models"

	"gorm.io/gorm"
)

// RegistrationData 注册数据结构
type RegistrationData struct {
	SlaveID int    `json:"slave_id"`
	IP      string `json:"ip"`
	Port    int    `json:"port"`
}

// HeartbeatData 心跳包数据结构
type HeartbeatData struct {
	SlaveID   int       `json:"slave_id"`
	Timestamp time.Time `json:"timestamp"`
}

// ConfigData 配置数据结构
type ConfigData struct {
	MqttHost string `json:"mqtt_host"`
	MqttPort int    `json:"mqtt_port"`
	Topic    string `json:"topic"`
	QoS      int    `json:"qos"`
	ClientID string `json:"client_id"`
	Start    int    `json:"start"`
	Step     int    `json:"step"`
	Command  string `json:"command"`   // 添加命令字段
	AckTopic string `json:"ack_topic"` // ACK主题配置
}

// ConfigResult 配置结果数据结构
type ConfigResult struct {
	SlaveID      int    `json:"slave_id"`
	SuccessCount int    `json:"success_count"`
	FailureCount int    `json:"failure_count"`
	Connections  int    `json:"connections"` // 添加连接数字段
	Message      string `json:"message"`
}

// Server master服务器结构
type Server struct {
	slaveModel *models.SlaveModel
	db         *gorm.DB
	server     *http.Server
	// 用于存储配置下发的结果
	configResults map[int]*ConfigResult
	resultsMutex  sync.RWMutex
}

// NewServer 创建新的master服务器实例
func NewServer() *Server {
	return &Server{
		slaveModel:    &models.SlaveModel{DB: db.DB},
		db:            db.DB,
		configResults: make(map[int]*ConfigResult),
	}
}

// Start 启动master服务器
func (s *Server) Start(ctx context.Context) {
	// 创建HTTP服务器
	mux := http.NewServeMux()
	mux.HandleFunc("/register", s.handleRegistration)
	mux.HandleFunc("/heartbeat", s.handleHeartbeat)
	mux.HandleFunc("/config-result", s.handleConfigResult)

	s.server = &http.Server{
		Addr:    ":8888",
		Handler: mux,
	}

	// 在后台启动服务器
	go func() {
		log.Println("Master server starting on port 8888")
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Master server error: %v", err)
		}

		// 监听上下文取消信号，用于优雅关闭服务器
		<-ctx.Done()
		log.Println("Shutting down master server...")
		if s.server != nil {
			s.server.Close()
		}
	}()

	// 启动定期检查slave状态的goroutine
	go s.checkSlaveStatus()

	log.Println("Master server started")
}

// checkSlaveStatus 定期检查slave状态
func (s *Server) checkSlaveStatus() {
	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for range ticker.C {
		// 检查数据库连接是否正常
		if s.db == nil {
			log.Println("Database connection is nil during status check, trying to reinitialize")
			db.InitDB()
			s.db = db.DB
			s.slaveModel = &models.SlaveModel{DB: db.DB}
		} else {
			// 检查数据库连接是否有效
			sqlDB, err := s.db.DB()
			if err != nil {
				log.Printf("Error getting sqlDB instance: %v, trying to reinitialize", err)
				db.InitDB()
				s.db = db.DB
				s.slaveModel = &models.SlaveModel{DB: db.DB}
			} else {
				// 尝试ping数据库连接
				if err := sqlDB.Ping(); err != nil {
					log.Printf("Database ping failed: %v, trying to reinitialize", err)
					db.InitDB()
					s.db = db.DB
					s.slaveModel = &models.SlaveModel{DB: db.DB}
				}
			}
		}

		// 获取所有slave
		slaves, err := s.slaveModel.GetAll()
		if err != nil {
			log.Printf("Error getting slaves for status check: %v", err)
			// 尝试重新初始化数据库连接
			db.InitDB()
			s.db = db.DB
			s.slaveModel = &models.SlaveModel{DB: db.DB}
			// 再次尝试获取slave列表
			slaves, err = s.slaveModel.GetAll()
			if err != nil {
				log.Printf("Error getting slaves after reinitialization: %v", err)
				continue
			}
		}

		log.Printf("Checking status for %d slaves", len(slaves))

		// 检查每个slave的状态
		for _, slave := range slaves {
			// 如果更新时间超过30秒，则认为slave离线
			// 给心跳包一些缓冲时间（心跳包每5秒发送一次）
			timeSinceUpdate := time.Since(slave.UpdatedAt)
			log.Printf("Slave %d: time since last update: %v, current status: %s", slave.ID, timeSinceUpdate, slave.Status)

			if timeSinceUpdate > 30*time.Second {
				if slave.Status != "offline" {
					// 只更新状态，不修改创建时间
					oldStatus := slave.Status
					slave.Status = "offline"
					slave.Connections = 0 // 将连接数归零
					err := s.slaveModel.UpdateWithoutConnections(slave)
					if err != nil {
						log.Printf("Error updating slave %d status to offline: %v", slave.ID, err)
						// 尝试重新初始化数据库连接后再次更新
						db.InitDB()
						s.db = db.DB
						s.slaveModel = &models.SlaveModel{DB: db.DB}
						err = s.slaveModel.UpdateWithoutConnections(slave)
						if err != nil {
							log.Printf("Error updating slave %d status to offline after reinitialization: %v", slave.ID, err)
						} else {
							log.Printf("Slave %d status updated from %s to offline after reinitialization", slave.ID, oldStatus)
						}
					} else {
						log.Printf("Slave %d status updated from %s to offline", slave.ID, oldStatus)
					}
				} else {
					log.Printf("Slave %d is already offline", slave.ID)
				}
			} else {
				log.Printf("Slave %d is still online (last update: %v ago)", slave.ID, timeSinceUpdate)
			}
		}
	}
}

// handleRegistration 处理slave注册请求
func (s *Server) handleRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析注册数据
	var regData RegistrationData
	if err := json.NewDecoder(r.Body).Decode(&regData); err != nil {
		log.Printf("Error decoding registration data: %v", err)
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	// 验证数据
	if regData.SlaveID <= 0 || regData.IP == "" || regData.Port <= 0 {
		log.Printf("Invalid registration data: SlaveID=%d, IP=%s, Port=%d", regData.SlaveID, regData.IP, regData.Port)
		http.Error(w, "Invalid registration data", http.StatusBadRequest)
		return
	}

	log.Printf("Received registration request from Slave %d at %s:%d", regData.SlaveID, regData.IP, regData.Port)

	// 检查数据库连接是否正常
	if s.db == nil {
		log.Println("Database connection is nil, trying to reinitialize")
		db.InitDB()
		s.db = db.DB
		s.slaveModel = &models.SlaveModel{DB: db.DB}
	}

	// 检查slave是否已存在
	existingSlave, err := s.slaveModel.GetByID(int64(regData.SlaveID))
	if err != nil {
		log.Printf("Error checking existing slave: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if existingSlave != nil {
		// 更新现有slave，但保持创建时间不变
		log.Printf("Updating existing slave %d", regData.SlaveID)
		log.Printf("Registration data: IP=%s, Port=%d", regData.IP, regData.Port)

		// 验证注册数据是否有效
		if regData.IP == "" {
			log.Printf("ERROR: Registration IP is empty for slave %d", regData.SlaveID)
			http.Error(w, "Invalid registration data: IP is empty", http.StatusBadRequest)
			return
		}

		if regData.Port <= 0 {
			log.Printf("ERROR: Registration Port is invalid (%d) for slave %d", regData.Port, regData.SlaveID)
			http.Error(w, "Invalid registration data: Port is invalid", http.StatusBadRequest)
			return
		}

		// 更新slave信息
		// SlaveHost和SlavePort存储slave自身的地址信息（来自注册数据）
		existingSlave.SlaveHost = regData.IP
		existingSlave.SlavePort = regData.Port
		// 如果MQTT地址信息尚未配置，则使用默认值
		if existingSlave.MqttHost == "" {
			existingSlave.MqttHost = "127.0.0.1" // 默认MQTT服务器地址
		}
		if existingSlave.MqttPort == 0 {
			existingSlave.MqttPort = 1883 // 默认MQTT端口
		}
		existingSlave.Status = "online" // 注册时设置为在线状态
		// 注意：不修改CreatedAt字段，保持为第一次注册的时间
		existingSlave.UpdatedAt = time.Now()

		log.Printf("Updating slave with data: ID=%d, Name=%s, MqttHost=%s, MqttPort=%d, SlaveHost=%s, SlavePort=%d, Status=%s",
			existingSlave.ID, existingSlave.Name, existingSlave.MqttHost, existingSlave.MqttPort,
			existingSlave.SlaveHost, existingSlave.SlavePort, existingSlave.Status)

		err = s.slaveModel.UpdateWithoutConnections(existingSlave)
		if err != nil {
			log.Printf("Error updating existing slave: %v", err)
			http.Error(w, "Failed to update slave data", http.StatusInternalServerError)
			return
		}
	} else {
		// 创建新slave
		log.Printf("Creating new slave %d", regData.SlaveID)
		log.Printf("Registration data: IP=%s, Port=%d", regData.IP, regData.Port)

		// 验证注册数据是否有效
		if regData.IP == "" {
			log.Printf("ERROR: Registration IP is empty for new slave %d", regData.SlaveID)
			http.Error(w, "Invalid registration data: IP is empty", http.StatusBadRequest)
			return
		}

		if regData.Port <= 0 {
			log.Printf("ERROR: Registration Port is invalid (%d) for new slave %d", regData.Port, regData.SlaveID)
			http.Error(w, "Invalid registration data: Port is invalid", http.StatusBadRequest)
			return
		}

		slave := &models.Slave{
			ID:        int64(regData.SlaveID),
			Name:      fmt.Sprintf("Slave-%d", regData.SlaveID),
			SlaveHost: regData.IP,   // Slave自身的地址信息（来自注册数据）
			SlavePort: regData.Port, // Slave自身的端口信息（来自注册数据）
			MqttHost:  "127.0.0.1",  // 默认MQTT服务器地址
			MqttPort:  1883,         // 默认MQTT端口
			Status:    "online",     // 新注册的slave设置为在线状态
		}

		log.Printf("Creating new slave with data: ID=%d, Name=%s, MqttHost=%s, MqttPort=%d, SlaveHost=%s, SlavePort=%d, Status=%s",
			slave.ID, slave.Name, slave.MqttHost, slave.MqttPort, slave.SlaveHost, slave.SlavePort, slave.Status)

		err = s.slaveModel.Insert(slave)
		if err != nil {
			log.Printf("Error creating new slave: %v", err)
			http.Error(w, "Failed to save slave data", http.StatusInternalServerError)
			return
		}
	}

	// 返回成功响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Registration successful"))
	log.Printf("Slave %d registered successfully from %s:%d", regData.SlaveID, regData.IP, regData.Port)
}

// handleHeartbeat 处理slave心跳包
func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析心跳数据
	var heartbeatData HeartbeatData
	if err := json.NewDecoder(r.Body).Decode(&heartbeatData); err != nil {
		log.Printf("Error decoding heartbeat data: %v", err)
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	// 验证数据
	if heartbeatData.SlaveID <= 0 {
		log.Printf("Invalid heartbeat data: SlaveID=%d", heartbeatData.SlaveID)
		http.Error(w, "Invalid heartbeat data", http.StatusBadRequest)
		return
	}

	log.Printf("Received heartbeat from Slave %d at %s", heartbeatData.SlaveID, heartbeatData.Timestamp)

	// 检查数据库连接是否正常
	if s.db == nil {
		log.Println("Database connection is nil during heartbeat, trying to reinitialize")
		db.InitDB()
		s.db = db.DB
		s.slaveModel = &models.SlaveModel{DB: db.DB}
	}

	// 更新slave时间戳和状态
	slave, err := s.slaveModel.GetByID(int64(heartbeatData.SlaveID))
	if err != nil {
		log.Printf("Error getting slave for heartbeat: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if slave == nil {
		log.Printf("Slave %d not found for heartbeat", heartbeatData.SlaveID)
		// 返回404错误，让slave知道需要重新注册
		http.Error(w, "Slave not found", http.StatusNotFound)
		return
	}

	// 更新时间戳和状态，但保持创建时间不变
	oldStatus := slave.Status
	slave.UpdatedAt = time.Now()
	slave.Status = "online" // 心跳包到达时更新状态为在线

	// 记录状态变化
	if oldStatus != slave.Status {
		log.Printf("Slave %d status changed from %s to %s", slave.ID, oldStatus, slave.Status)
	}

	err = s.slaveModel.UpdateWithoutConnections(slave)
	if err != nil {
		log.Printf("Error updating slave for heartbeat: %v", err)
		http.Error(w, "Failed to update slave", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Heartbeat received"))
	log.Printf("Heartbeat processed successfully for slave %d, status: %s, updated_at: %v", heartbeatData.SlaveID, slave.Status, slave.UpdatedAt)
}

// handleConfigResult 处理slave配置结果反馈
func (s *Server) handleConfigResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析配置结果数据
	var configResult ConfigResult
	if err := json.NewDecoder(r.Body).Decode(&configResult); err != nil {
		log.Printf("Error decoding config result data: %v", err)
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	// 验证数据
	if configResult.SlaveID <= 0 {
		log.Printf("Invalid config result data: SlaveID=%d", configResult.SlaveID)
		http.Error(w, "Invalid config result data", http.StatusBadRequest)
		return
	}

	log.Printf("Received config result from Slave %d: Success=%d, Failure=%d, Connections=%d, Message=%s",
		configResult.SlaveID, configResult.SuccessCount, configResult.FailureCount, configResult.Connections, configResult.Message)

	// 存储配置结果
	s.resultsMutex.Lock()
	s.configResults[configResult.SlaveID] = &configResult
	s.resultsMutex.Unlock()

	// 更新slave的连接数
	slave, err := s.slaveModel.GetByID(int64(configResult.SlaveID))
	if err != nil {
		log.Printf("Error getting slave %d: %v", configResult.SlaveID, err)
	} else if slave != nil {
		// 更新连接数
		slave.Connections = configResult.Connections
		// 更新时间戳
		slave.UpdatedAt = time.Now()
		// 更新状态为运行中（如果有成功连接）
		if configResult.SuccessCount > 0 {
			slave.Status = "running"
		}
		err = s.slaveModel.UpdateWithConnections(slave)
		if err != nil {
			log.Printf("Error updating slave %d connections: %v", configResult.SlaveID, err)
		} else {
			log.Printf("Slave %d connections updated to %d", configResult.SlaveID, configResult.Connections)
		}
	}

	// 返回成功响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Config result received"))
	log.Printf("Config result processed successfully for slave %d", configResult.SlaveID)
}

// deployConfigToSlave 向单个slave下发配置
func (s *Server) deployConfigToSlave(slaveID int64) error {
	// 获取slave信息
	slave, err := s.slaveModel.GetByID(slaveID)
	if err != nil {
		return fmt.Errorf("error getting slave %d: %v", slaveID, err)
	}

	if slave == nil {
		return fmt.Errorf("slave %d not found", slaveID)
	}

	// 构造配置数据
	configData := ConfigData{
		MqttHost: slave.MqttHost,
		MqttPort: slave.MqttPort,
		Topic:    slave.Topic,
		QoS:      slave.QoS,
		ClientID: slave.ClientID,
		Start:    slave.Start,
		Step:     slave.Step,
		AckTopic: slave.AckTopic, // 使用配置的ACK主题，如果为空则使用默认值
	}

	// 添加调试日志，查看下发的配置数据
	log.Printf("下发配置数据到Slave %d: ClientID=%s, Start=%d, Step=%d", slaveID, slave.ClientID, slave.Start, slave.Step)

	// 构造消息结构
	message := struct {
		Type    string     `json:"type"`
		Content ConfigData `json:"content"`
	}{
		Type:    "config",
		Content: configData,
	}

	// 将消息序列化为JSON
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %v", err)
	}

	// 构造slave的配置URL，使用net.JoinHostPort来正确处理IPv4和IPv6地址
	configURL := net.JoinHostPort(slave.SlaveHost, fmt.Sprintf("%d", slave.SlavePort))

	// 创建TCP连接
	conn, err := net.DialTimeout("tcp", configURL, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to slave %d at %s: %v", slaveID, configURL, err)
	}
	defer conn.Close()

	// 发送JSON数据并在末尾添加换行符
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to send config to slave %d: %v", slaveID, err)
	}

	// 确保数据被刷新到网络
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.CloseWrite()
	}

	// 等待更长时间确保数据被接收和处理
	time.Sleep(500 * time.Millisecond)

	log.Printf("Config deployed successfully to slave %d at %s", slaveID, configURL)
	return nil
}

// DeployConfigToSlaves 下发配置到指定的Slaves
func (s *Server) DeployConfigToSlaves(slaveIDs []int64) error {
	for _, slaveID := range slaveIDs {
		err := s.deployConfigToSlave(slaveID)
		if err != nil {
			log.Printf("Failed to deploy config to slave %d: %v", slaveID, err)
			// 继续处理其他slave，不中断整个过程
		}
	}
	return nil
}

// GetSlaveModel 获取slave模型实例
func (s *Server) GetSlaveModel() *models.SlaveModel {
	return s.slaveModel
}

// GetAllSlaves 获取所有slave记录（用于调试）
func (s *Server) GetAllSlaves() ([]*models.Slave, error) {
	log.Println("Getting all slaves from database")
	slaves, err := s.slaveModel.GetAll()
	if err != nil {
		log.Printf("Error getting slaves from database: %v", err)
		return nil, err
	}
	log.Printf("Retrieved %d slaves from database", len(slaves))

	// 添加额外的日志来帮助调试
	for i, slave := range slaves {
		log.Printf("Slave %d: ID=%d, Name=%s, Status=%s, SlaveHost=%s, SlavePort=%d, MqttHost=%s, MqttPort=%d",
			i, slave.ID, slave.Name, slave.Status, slave.SlaveHost, slave.SlavePort, slave.MqttHost, slave.MqttPort)
	}

	// 确保返回的slave列表不为nil
	if slaves == nil {
		log.Println("GetAllSlaves returning empty slice instead of nil")
		slaves = make([]*models.Slave, 0)
	}

	return slaves, nil
}

// GetConfigResult 获取指定slave的配置结果
func (s *Server) GetConfigResult(slaveID int) *ConfigResult {
	s.resultsMutex.RLock()
	defer s.resultsMutex.RUnlock()

	result, exists := s.configResults[slaveID]
	if !exists {
		return nil
	}

	// 返回结果的副本，避免外部修改
	resultCopy := *result
	return &resultCopy
}

// ClearConfigResult 清除指定slave的配置结果
func (s *Server) ClearConfigResult(slaveID int) {
	s.resultsMutex.Lock()
	defer s.resultsMutex.Unlock()

	delete(s.configResults, slaveID)
}
