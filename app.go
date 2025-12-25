package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"mqttbench/internal/db"
	"mqttbench/internal/master"
	"mqttbench/internal/message"
	"mqttbench/internal/models"
	"mqttbench/internal/performance"

	"gorm.io/gorm"
)

// App struct
type App struct {
	ctx                context.Context
	masterServer       *master.Server
	performanceService *performance.Service
	messageService     *message.Service
	db                 *gorm.DB
}

// NewApp creates a new App application struct
func NewApp() *App {
	// 初始化数据库
	db.InitDB()

	app := &App{
		db: db.DB,
	}

	// 创建master服务器实例
	app.masterServer = master.NewServer()

	// 创建性能测试服务实例
	app.performanceService = performance.NewService()

	// 创建消息测试服务实例
	app.messageService = message.NewService()

	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 启动master服务器
	a.masterServer.Start(ctx)
}

// GetSlaves 获取所有Slave
func (a *App) GetSlaves() ([]*models.Slave, error) {
	log.Println("GetSlaves method called")

	// 检查数据库连接是否正常
	if a.db == nil {
		log.Println("Database connection is nil in GetSlaves, trying to reinitialize")
		db.InitDB()
		a.db = db.DB
		a.masterServer = master.NewServer()
	} else {
		// 检查数据库连接是否有效
		sqlDB, err := a.db.DB()
		if err != nil {
			log.Printf("Error getting sqlDB instance in GetSlaves: %v, trying to reinitialize", err)
			db.InitDB()
			a.db = db.DB
			a.masterServer = master.NewServer()
		} else {
			// 尝试ping数据库连接
			if err := sqlDB.Ping(); err != nil {
				log.Printf("Database ping failed in GetSlaves: %v, trying to reinitialize", err)
				db.InitDB()
				a.db = db.DB
				a.masterServer = master.NewServer()
			}
		}
	}

	// 先尝试使用调试方法获取slave列表
	slaves, err := a.masterServer.GetAllSlaves()
	if err != nil {
		log.Printf("GetAllSlaves failed: %v, falling back to GetAll", err)
		// 如果调试方法出错，回退到原来的方法
		slaves, err = a.masterServer.GetSlaveModel().GetAll()
		if err != nil {
			log.Printf("GetAll also failed: %v", err)
			return nil, err
		}
	}

	// 添加额外的日志来帮助调试
	log.Printf("GetSlaves returning %d slaves", len(slaves))
	for i, slave := range slaves {
		log.Printf("Slave %d: ID=%d, Name=%s, Status=%s, SlaveHost=%s, SlavePort=%d, MqttHost=%s, MqttPort=%d",
			i, slave.ID, slave.Name, slave.Status, slave.SlaveHost, slave.SlavePort, slave.MqttHost, slave.MqttPort)
	}

	// 确保返回的slave列表不为nil
	if slaves == nil {
		log.Println("GetSlaves returning empty slice instead of nil")
		slaves = make([]*models.Slave, 0)
	}

	return slaves, nil
}

// AddSlave 添加Slave
func (a *App) AddSlave(name string, mqttHost string, mqttPort int, clientID string, topic string, qos int, start int, step int, ackTopic string) (*models.Slave, error) {
	// 为MQTT主机和端口设置默认值
	if mqttHost == "" {
		mqttHost = "127.0.0.1" // 默认本地MQTT服务器
	}

	if mqttPort <= 0 {
		mqttPort = 1883 // MQTT默认端口
	}

	// 如果ACK主题为空，则使用默认值
	if ackTopic == "" {
		ackTopic = "EEW/ACK/Channel1"
	}

	slave := &models.Slave{
		Name:     name,
		MqttHost: mqttHost,
		MqttPort: mqttPort,
		ClientID: clientID,
		Topic:    topic,
		QoS:      qos,
		Start:    start,
		Step:     step,
		AckTopic: ackTopic,
		Status:   "offline", // 新增的slave默认为离线状态
	}

	err := a.masterServer.GetSlaveModel().Insert(slave)
	if err != nil {
		return nil, err
	}

	return slave, nil
}

// UpdateSlave 更新Slave
func (a *App) UpdateSlave(id int64, name string, mqttHost string, mqttPort int, clientID string, topic string, qos int, start int, step int, ackTopic string) error {
	// 首先获取现有的slave信息
	existingSlave, err := a.masterServer.GetSlaveModel().GetByID(id)
	if err != nil {
		return err
	}

	// 如果不存在该slave，则返回错误
	if existingSlave == nil {
		return gorm.ErrRecordNotFound
	}

	// 如果字段为空，则保持原有值
	if name == "" {
		name = existingSlave.Name
	}

	if mqttHost == "" {
		mqttHost = existingSlave.MqttHost
	}

	if mqttPort == 0 {
		mqttPort = existingSlave.MqttPort
	}

	if clientID == "" {
		clientID = existingSlave.ClientID
	}

	if topic == "" {
		topic = existingSlave.Topic
	}

	// 如果ACK主题为空，则保持原有值
	if ackTopic == "" {
		ackTopic = existingSlave.AckTopic
	}

	// 对于QoS、Start、Step，如果传入的是-1，则保持原有值
	// 因为0是QoS的有效值，所以我们需要特殊处理
	if qos == -1 {
		qos = existingSlave.QoS
	}

	if start == -1 {
		start = existingSlave.Start
	}

	if step == -1 {
		step = existingSlave.Step
	}

	slave := &models.Slave{
		ID:        id,
		Name:      name,
		MqttHost:  mqttHost,
		MqttPort:  mqttPort,
		SlaveHost: existingSlave.SlaveHost, // 保持原有的SlaveHost
		SlavePort: existingSlave.SlavePort, // 保持原有的SlavePort
		ClientID:  clientID,
		Topic:     topic,
		QoS:       qos,
		Start:     start,
		Step:      step,
		AckTopic:  ackTopic,             // 更新ACK主题
		Status:    existingSlave.Status, // 保持原有的状态
	}

	return a.masterServer.GetSlaveModel().UpdateWithoutConnections(slave)
}

// DeleteSlave 删除Slave
func (a *App) DeleteSlave(id int64) error {
	return a.masterServer.GetSlaveModel().Delete(id)
}

// DeployConfig 下发配置到指定的Slave
func (a *App) DeployConfig(slaveIDs []int64) error {
	return a.masterServer.DeployConfigToSlaves(slaveIDs)
}

// StartSlave 启动指定的Slave
func (a *App) StartSlave(slaveID int64) error {
	// 首先获取slave信息
	slave, err := a.masterServer.GetSlaveModel().GetByID(slaveID)
	if err != nil {
		return err
	}

	if slave == nil {
		return fmt.Errorf("slave %d not found", slaveID)
	}

	// 构造带有启动命令的配置数据
	configData := master.ConfigData{
		MqttHost: slave.MqttHost,
		MqttPort: slave.MqttPort,
		Topic:    slave.Topic,
		QoS:      slave.QoS,
		ClientID: slave.ClientID,
		Start:    slave.Start,
		Step:     slave.Step,
		Command:  "start",        // 添加启动命令
		AckTopic: slave.AckTopic, // 添加ACK主题
	}

	// 构造消息结构
	message := struct {
		Type    string            `json:"type"`
		Content master.ConfigData `json:"content"`
	}{
		Type:    "config",
		Content: configData,
	}

	// 将消息序列化为JSON
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %v", err)
	}

	// 添加调试日志
	log.Printf("Sending config data to slave %d: %s", slaveID, string(data))

	// 构造slave的配置URL
	configURL := net.JoinHostPort(slave.SlaveHost, fmt.Sprintf("%d", slave.SlavePort))

	// 创建TCP连接
	conn, err := net.DialTimeout("tcp", configURL, 10*time.Second)
	if err != nil {
		// 更新slave状态为离线
		slave.Status = "offline"
		updateErr := a.masterServer.GetSlaveModel().UpdateWithoutConnections(slave)
		if updateErr != nil {
			log.Printf("Failed to update slave status to offline: %v", updateErr)
		}
		return fmt.Errorf("failed to connect to slave %d at %s: %v", slaveID, configURL, err)
	}
	defer conn.Close()

	// 发送JSON数据并在末尾添加换行符
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		// 更新slave状态为离线
		slave.Status = "offline"
		updateErr := a.masterServer.GetSlaveModel().UpdateWithoutConnections(slave)
		if updateErr != nil {
			log.Printf("Failed to update slave status to offline: %v", updateErr)
		}
		return fmt.Errorf("failed to send config to slave %d: %v", slaveID, err)
	}

	// 确保数据被刷新到网络
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.CloseWrite()
	}

	// 等待更长时间确保数据被接收和处理
	time.Sleep(500 * time.Millisecond)

	// 更新slave状态为运行中
	slave.Status = "running"
	updateErr := a.masterServer.GetSlaveModel().UpdateWithoutConnections(slave)
	if updateErr != nil {
		log.Printf("Failed to update slave status to running: %v", updateErr)
	}

	log.Printf("Start command deployed successfully to slave %d at %s", slaveID, configURL)
	return nil
}

// StopSlave 停止指定的Slave
func (a *App) StopSlave(slaveID int64) error {
	log.Printf("StopSlave方法被调用，slaveID: %d", slaveID)
	// 首先获取slave信息
	slave, err := a.masterServer.GetSlaveModel().GetByID(slaveID)
	if err != nil {
		log.Printf("获取slave信息失败: %v", err)
		return err
	}

	if slave == nil {
		log.Printf("未找到slave %d", slaveID)
		return fmt.Errorf("slave %d not found", slaveID)
	}

	log.Printf("获取到slave信息: ID=%d, Name=%s, SlaveHost=%s, SlavePort=%d", slave.ID, slave.Name, slave.SlaveHost, slave.SlavePort)

	// 构造带有停止命令的配置数据
	configData := master.ConfigData{
		Command: "stop", // 添加停止命令
	}

	// 构造消息结构
	message := struct {
		Type    string            `json:"type"`
		Content master.ConfigData `json:"content"`
	}{
		Type:    "config",
		Content: configData,
	}

	// 将消息序列化为JSON
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化配置数据失败: %v", err)
		return fmt.Errorf("failed to marshal config data: %v", err)
	}

	// 添加调试日志
	log.Printf("发送停止命令到slave %d: %s", slaveID, string(data))

	// 构造slave的配置URL
	configURL := net.JoinHostPort(slave.SlaveHost, fmt.Sprintf("%d", slave.SlavePort))
	log.Printf("构造的配置URL: %s", configURL)

	// 创建TCP连接
	log.Printf("尝试连接到slave %d at %s", slaveID, configURL)
	conn, err := net.DialTimeout("tcp", configURL, 10*time.Second)
	if err != nil {
		log.Printf("连接到slave失败: %v", err)
		// 更新slave状态为离线
		slave.Status = "offline"
		slave.Connections = 0 // 将连接数归零
		updateErr := a.masterServer.GetSlaveModel().UpdateWithoutConnections(slave)
		if updateErr != nil {
			log.Printf("更新slave状态为离线失败: %v", updateErr)
		}
		return fmt.Errorf("failed to connect to slave %d at %s: %v", slaveID, configURL, err)
	}
	defer conn.Close()
	log.Printf("成功连接到slave %d", slaveID)

	// 发送JSON数据并在末尾添加换行符
	log.Printf("发送数据到slave: %s", string(data))
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		log.Printf("发送配置到slave失败: %v", err)
		// 更新slave状态为离线
		slave.Status = "offline"
		slave.Connections = 0 // 将连接数归零
		updateErr := a.masterServer.GetSlaveModel().UpdateWithoutConnections(slave)
		if updateErr != nil {
			log.Printf("更新slave状态为离线失败: %v", updateErr)
		}
		return fmt.Errorf("failed to send config to slave %d: %v", slaveID, err)
	}
	log.Printf("成功发送数据到slave %d", slaveID)

	// 确保数据被刷新到网络
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		log.Printf("关闭TCP写入端")
		tcpConn.CloseWrite()
	}

	// 等待更长时间确保数据被接收和处理
	log.Printf("等待500ms确保数据被接收和处理")
	time.Sleep(500 * time.Millisecond)

	// 更新slave状态为离线
	slave.Status = "offline"
	slave.Connections = 0 // 将连接数归零
	updateErr := a.masterServer.GetSlaveModel().UpdateWithoutConnections(slave)
	if updateErr != nil {
		log.Printf("更新slave状态为离线失败: %v", updateErr)
	}

	log.Printf("停止命令成功部署到slave %d at %s", slaveID, configURL)
	return nil
}

// GetConfigResult 获取指定Slave的配置结果
func (a *App) GetConfigResult(slaveID int) *master.ConfigResult {
	return a.masterServer.GetConfigResult(slaveID)
}

// ClearConfigResult 清除指定Slave的配置结果
func (a *App) ClearConfigResult(slaveID int) {
	a.masterServer.ClearConfigResult(slaveID)
}

// GetPerformanceTests 获取所有性能测试记录
func (a *App) GetPerformanceTests() ([]*models.Performance, error) {
	return a.performanceService.GetPerformanceTests()
}

// GetMessageTests 获取所有消息测试记录
func (a *App) GetMessageTests() ([]*models.Message, error) {
	return a.messageService.GetMessageTests()
}
