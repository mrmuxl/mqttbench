package slave

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// 定义停止函数类型
type StopFunc func()

// 用于存储停止函数的全局变量
var (
	stopSlaveWithoutStatusChangeFunc StopFunc
	stopFuncMutex                    sync.RWMutex
)

// SetMasterInfo 设置Master连接信息
func SetMasterInfo(ip string, port int, id int) {
	// 移除未使用的masterPort变量赋值
	// masterPort = port
}

// SetStopFunc 设置停止函数
func SetStopFunc(stopFunc StopFunc) {
	log.Printf("SetStopFunc被调用，设置停止函数")
	stopFuncMutex.Lock()
	defer stopFuncMutex.Unlock()

	stopSlaveWithoutStatusChangeFunc = stopFunc
}

// Message 定义消息结构
type Message struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

// ConfigData 配置数据结构
type ConfigData struct {
	MqttHost string `json:"mqtt_host"`
	MqttPort int    `json:"mqtt_port"`
	Command  string `json:"command"` // 添加命令字段
	Topic    string `json:"topic"`
	QoS      int    `json:"qos"`
	ClientID string `json:"client_id"`
	Start    int    `json:"start"`
	Step     int    `json:"step"`
	AckTopic string `json:"ack_topic"` // ACK主题配置
}

// StartSlaveServer 启动slave服务器，监听随机端口
func StartSlaveServer(configChan chan<- ConfigData) (int, chan Message, error) {
	// 监听随机端口
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to start server: %v", err)
	}

	// 获取实际分配的端口号
	port := listener.Addr().(*net.TCPAddr).Port

	// 创建用于传递接收到的消息的通道
	messageChan := make(chan Message, 10)

	// 在后台启动服务器
	go func() {
		defer listener.Close()
		log.Printf("Slave server listening on port %d", port)

		for {
			// 接受连接
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Error accepting connection: %v", err)
				continue
			}

			// 在新的goroutine中处理连接
			go handleConnection(conn, messageChan, configChan)
		}
	}()

	return port, messageChan, nil
}

// handleConnection 处理客户端连接
func handleConnection(conn net.Conn, messageChan chan<- Message, configChan chan<- ConfigData) {
	defer conn.Close()
	log.Printf("新的连接来自: %s", conn.RemoteAddr().String())

	// 创建解码器
	decoder := json.NewDecoder(conn)

	// 设置解码器选项
	decoder.UseNumber()

	// 循环读取JSON消息
	for {
		// 首先尝试解析为通用消息格式
		var msg Message
		log.Printf("等待接收消息...")
		if err := decoder.Decode(&msg); err != nil {
			// 检查是否是EOF错误（连接正常关闭）
			if err == io.EOF {
				log.Printf("连接被客户端关闭: %s", conn.RemoteAddr().String())
				return
			}

			// 检查是否是网络错误
			if netErr, ok := err.(*net.OpError); ok {
				log.Printf("网络错误: %v", netErr)
				return
			}

			// 其他解码错误
			log.Printf("解码消息错误: %v", err)
			return
		}

		log.Printf("接收到消息: Type=%s, Content=%v", msg.Type, msg.Content)

		// 检查消息类型
		if msg.Type == "config" {
			// 添加调试日志
			log.Printf("Received config message: %+v", msg)

			// 如果是配置消息，尝试解析为配置数据
			if contentBytes, err := json.Marshal(msg.Content); err == nil {
				var configData ConfigData
				if err := json.Unmarshal(contentBytes, &configData); err == nil {
					// 检查是否有启动命令
					if configData.Command == "start" {
						log.Printf("Received start command")
					}

					// 检查是否有停止命令
					if configData.Command == "stop" {
						log.Printf("Received stop command")
						// 处理停止命令
						handleStopCommand()
					} else {
						log.Printf("Received config command: %s", configData.Command)
					}

					// 将配置数据发送到配置通道
					configChan <- configData
					log.Printf("Received config update: %+v", configData)
				} else {
					log.Printf("Error parsing config data: %v", err)
				}
			}
		} else {
			// 将其他类型的消息发送到消息通道
			messageChan <- msg
			log.Printf("Received message: Type=%s, Content=%v", msg.Type, msg.Content)
		}
	}
}

// handleStopCommand 处理停止命令
func handleStopCommand() {
	// 断开所有MQTT连接
	// 发送状态更新到Master，连接数为0，但不改变Slave状态
	log.Println("Slave已停止，所有连接已断开")

	stopFuncMutex.RLock()
	stopFunc := stopSlaveWithoutStatusChangeFunc
	stopFuncMutex.RUnlock()

	log.Println("调用停止函数前，stopSlaveWithoutStatusChangeFunc是否为nil:", stopFunc == nil)

	// 调用停止函数，但不改变Slave状态
	if stopFunc != nil {
		stopFunc()
	} else {
		log.Println("警告：stopSlaveWithoutStatusChangeFunc为nil，无法调用停止函数")
	}
}
