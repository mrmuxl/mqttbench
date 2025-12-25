package slave

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// 用于跟踪连接数的全局变量
var (
	connectionCount      int64
	expectedConnections  int64
	messageCount         int64 // 添加消息计数器
	ackMessageCount      int64 // 添加ACK消息计数器
	connectionMutex      sync.RWMutex
	onConnectionComplete func(successCount int) // 所有连接完成时的回调函数
)

// SetConnectionCompleteCallback 设置所有连接完成时的回调函数
func SetConnectionCompleteCallback(callback func(successCount int)) {
	connectionMutex.Lock()
	onConnectionComplete = callback
	connectionMutex.Unlock()
}

// SetExpectedConnections 设置期望的连接数
func SetExpectedConnections(count int) {
	atomic.StoreInt64(&expectedConnections, int64(count))
	atomic.StoreInt64(&connectionCount, 0)
}

// incrementConnectionCount 增加连接计数
func incrementConnectionCount() {
	newCount := atomic.AddInt64(&connectionCount, 1)
	expected := atomic.LoadInt64(&expectedConnections)

	// 检查是否所有连接都已完成
	if newCount == expected && expected > 0 {
		// 调用连接完成回调函数
		connectionMutex.RLock()
		callback := onConnectionComplete
		connectionMutex.RUnlock()

		if callback != nil {
			callback(int(newCount))
		}
	}
}

// GetMessageCount 获取消息计数
func GetMessageCount() int64 {
	return atomic.LoadInt64(&messageCount)
}

// GetConnectionCount 获取当前连接数
func GetConnectionCount() int {
	return int(atomic.LoadInt64(&connectionCount))
}

// ResetConnectionCount 重置连接计数
func ResetConnectionCount() {
	atomic.StoreInt64(&connectionCount, 0)
	atomic.StoreInt64(&expectedConnections, 0)
}

// ResetMessageCount 重置消息计数
func ResetMessageCount() {
	atomic.StoreInt64(&messageCount, 0)
}

// GetAckMessageCount 获取ACK消息计数
func GetAckMessageCount() int64 {
	return atomic.LoadInt64(&ackMessageCount)
}

// ResetAckMessageCount 重置ACK消息计数
func ResetAckMessageCount() {
	atomic.StoreInt64(&ackMessageCount, 0)
}

// MQTTClient 封装MQTT客户端
type MQTTClient struct {
	client   mqtt.Client
	config   ConfigData
	topic    string       // 用于存储订阅的主题
	qos      byte         // 用于存储订阅的QoS
	ackTopic string       // 用于存储ACK主题
	mutex    sync.RWMutex // 用于保护客户端状态的互斥锁
}

// NewMQTTClient 创建新的MQTT客户端
func NewMQTTClient(config ConfigData) *MQTTClient {
	// 如果配置中没有设置ACK主题，则使用默认值
	ackTopic := config.AckTopic
	if ackTopic == "" {
		ackTopic = "EEW/ACK/Channel1"
	}

	return &MQTTClient{
		config:   config,
		ackTopic: ackTopic,
	}
}

// Connect 连接到MQTT服务器
func (m *MQTTClient) Connect(clientID string) error {
	// 构造MQTT服务器地址
	broker := fmt.Sprintf("tcp://%s:%d", m.config.MqttHost, m.config.MqttPort)

	// 设置MQTT客户端选项
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)

	// 设置用户名和密码为clientID，满足username=password=clientID的要求
	opts.SetClientID(clientID)
	opts.SetUsername(clientID)
	opts.SetPassword(clientID)

	// 设置其他选项
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(10 * time.Second)
	opts.SetKeepAlive(120 * time.Second)

	// 设置TLS配置（如果需要）
	if m.config.MqttPort == 8883 {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	}

	// 设置连接和断开连接的回调
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		// 如果已有主题信息，则自动订阅
		m.mutex.RLock()
		topic := m.topic
		qos := m.qos
		m.mutex.RUnlock()

		if topic != "" {
			err := m.Subscribe(topic, qos, clientID)
			if err != nil {
				log.Printf("MQTT客户端 %s 自动订阅主题 %s 失败: %v", clientID, topic, err)
			}
		}
		// 增加连接计数（会在所有连接完成时触发回调）
		incrementConnectionCount()
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("MQTT客户端 %s 连接丢失: %v", clientID, err)

		m.mutex.RLock()
		topic := m.topic
		qos := m.qos
		m.mutex.RUnlock()

		if topic != "" {
			err = m.Subscribe(topic, qos, clientID)
			if err != nil {
				log.Printf("MQTT客户端 %s 自动订阅主题 %s 失败: %v", clientID, topic, err)
			} else {
				log.Printf("MQTT客户端 %s 掉线后自动订阅主题 %s 成功", clientID, topic)
			}
		}
	})

	// 创建MQTT客户端
	client := mqtt.NewClient(opts)

	// 安全地设置客户端实例
	m.mutex.Lock()
	m.client = client
	m.mutex.Unlock()

	// 连接到MQTT服务器
	token := client.Connect()
	if !token.WaitTimeout(120 * time.Second) {
		return fmt.Errorf("连接到MQTT服务器超时")
	}

	if token.Error() != nil {
		return fmt.Errorf("连接到MQTT服务器失败: %v", token.Error())
	}

	// log.Printf("MQTT客户端 %s 连接成功到 %s", clientID, broker)
	return nil
}

// Subscribe 订阅主题
func (m *MQTTClient) Subscribe(topic string, qos byte, clientID string) error {
	m.mutex.RLock()
	client := m.client
	m.mutex.RUnlock()

	if client == nil {
		return fmt.Errorf("MQTT客户端未连接")
	}

	// 保存主题信息和QoS，用于重连后重新订阅
	m.mutex.Lock()
	m.topic = topic
	m.qos = qos
	m.mutex.Unlock()

	token := client.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
		// 增加消息计数器
		newCount := atomic.AddInt64(&messageCount, 1)
		log.Printf("收到消息总数: %d,", newCount)

		// 使用回调函数处理消息并发送ACK确认
		go m.handleMessageWithACK(msg, clientID)
	})

	if !token.WaitTimeout(60 * time.Second) {
		return fmt.Errorf("订阅主题 %s 超时", topic)
	}

	if token.Error() != nil {
		return fmt.Errorf("订阅主题 %s 失败: %v", topic, token.Error())
	}

	// log.Printf("成功订阅主题: %s, QoS: %d", topic, qos)
	return nil
}

// handleMessageWithACK 处理消息并发送ACK确认
func (m *MQTTClient) handleMessageWithACK(msg mqtt.Message, clientID string) {
	// 解析JSON数据
	var jsonData map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &jsonData); err != nil {
		log.Printf("解析JSON消息失败: %v", err)
		return
	}

	// log.Printf("解析JSON消息成功: %+v", jsonData)

	// 根据接收的数据构造ACK消息
	ackData := constructACKData(jsonData, clientID)

	// 将ACK数据序列化为JSON
	ackPayload, err := json.Marshal(ackData)
	if err != nil {
		log.Printf("序列化ACK消息失败: %v", err)
		return
	}

	// 使用配置的ACK主题发布确认消息
	ackTopic := m.ackTopic

	// 发布ACK消息
	token := m.client.Publish(ackTopic, msg.Qos(), false, ackPayload)
	if !token.WaitTimeout(120 * time.Second) {
		log.Printf("发布ACK消息到主题 %s 超时", ackTopic)
		return
	}

	if token.Error() != nil {
		log.Printf("发布ACK消息到主题 %s 失败: %v", ackTopic, token.Error())
		return
	}

	// 增加ACK消息计数器
	newAckCount := atomic.AddInt64(&ackMessageCount, 1)
	log.Printf("发布ACK消息总数: %d,", newAckCount)
}

// constructACKData 根据接收的数据构造ACK响应数据
func constructACKData(receivedData map[string]interface{}, clientID string) map[string]interface{} {
	now := time.Now().Format("2006-01-02 15:04:05.999")
	// 构造ACK消息，根据接收数据的字段映射到响应数据
	ackData := map[string]interface{}{
		"1":  getOrDefault(receivedData, "1", ""),
		"2":  getOrDefault(receivedData, "2", ""),
		"3":  now,                                 //接收时间
		"4":  now,                                 //发送时间
		"5":  clientID,                            // 使用客户端ID
	}

	return ackData
}

// getOrDefault 获取map中的值，如果不存在则返回默认值
func getOrDefault(data map[string]interface{}, key string, defaultValue interface{}) interface{} {
	if value, exists := data[key]; exists {
		return value
	}
	return defaultValue
}

// // SubscribeWithCallback 使用自定义回调函数订阅主题
// func (m *MQTTClient) SubscribeWithCallback(topic string, qos byte, callback mqtt.MessageHandler) error {
// 	if m.client == nil {
// 		return fmt.Errorf("MQTT客户端未连接")
// 	}

// 	token := m.client.Subscribe(topic, qos, callback)

// 	if !token.WaitTimeout(30 * time.Second) {
// 		return fmt.Errorf("订阅主题 %s 超时", topic)
// 	}

// 	if token.Error() != nil {
// 		return fmt.Errorf("订阅主题 %s 失败: %v", topic, token.Error())
// 	}

// 	log.Printf("成功订阅主题: %s, QoS: %d", topic, qos)
// 	return nil
// }

// Publish 发布消息
func (m *MQTTClient) Publish(topic string, qos byte, payload interface{}) error {
	m.mutex.RLock()
	client := m.client
	m.mutex.RUnlock()

	if client == nil {
		return fmt.Errorf("MQTT客户端未连接")
	}

	token := client.Publish(topic, qos, false, payload)
	if !token.WaitTimeout(30 * time.Second) {
		return fmt.Errorf("发布消息到主题 %s 超时", topic)
	}

	if token.Error() != nil {
		return fmt.Errorf("发布消息到主题 %s 失败: %v", topic, token.Error())
	}

	log.Printf("成功发布消息到主题: %s, QoS: %d", topic, qos)
	return nil
}

// Disconnect 断开MQTT连接
func (m *MQTTClient) Disconnect() {
	log.Println("MQTTClient.Disconnect() 方法被调用")
	m.mutex.RLock()
	client := m.client
	m.mutex.RUnlock()

	if client != nil && client.IsConnected() {
		log.Println("MQTT客户端已连接，正在断开连接")
		client.Disconnect(250)
		log.Printf("MQTT客户端已断开连接")
	} else {
		log.Println("MQTT客户端未连接或client为nil")
	}
}

// IsConnected 检查MQTT客户端是否连接
func (m *MQTTClient) IsConnected() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.client != nil && m.client.IsConnected()
}

// GetTopic 获取订阅的主题
func (m *MQTTClient) GetTopic() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.topic
}

// GetQoS 获取订阅的QoS
func (m *MQTTClient) GetQoS() byte {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.qos
}
