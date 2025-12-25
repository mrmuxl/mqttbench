package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
	"sync"
	"time"

	"mqttbench/internal/slave"
)

// Version information set at build time
var (
	Version   string
	BuildTime string
)

// 用于存储活跃的MQTT客户端
var (
	activeClients = make(map[string]*slave.MQTTClient)
	clientsMutex  = sync.RWMutex{}
)

// 用于存储Master连接信息
var (
	masterIP   string
	masterPort int
	slaveID    int
)

// ConfigResult 配置结果数据结构
type ConfigResult struct {
	SlaveID      int    `json:"slave_id"`
	SuccessCount int    `json:"success_count"`
	FailureCount int    `json:"failure_count"`
	Connections  int    `json:"connections"` // 添加连接数字段
	Message      string `json:"message"`
}

func main() {
	// 定义命令行参数
	masterIPFlag := flag.String("ip", "127.0.0.1", "Master IP地址")
	masterPortFlag := flag.Int("port", 8888, "Master端口号")
	pprofPortFlag := flag.Int("pprof-port", 6060, "pprof端口号")
	versionFlag := flag.Bool("version", false, "显示版本信息")

	flag.Parse()

	// 检查是否请求版本信息
	if *versionFlag {
		fmt.Printf("Slave Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		return
	}

	// 检查是否提供了必要参数
	if *masterIPFlag == "" || *masterPortFlag <= 0 {
		fmt.Println("请提供Master IP地址和端口号")
		fmt.Println("用法: slave -ip=IP地址 -port=端口号")
		fmt.Println("示例: slave -ip=192.168.1.100 -port=1883")
		return
	}

	// 启动pprof服务器
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		log.Printf("pprof服务器启动在端口 %d", *pprofPortFlag)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *pprofPortFlag), mux))
	}()

	// 设置全局变量
	masterIP = *masterIPFlag
	masterPort = *masterPortFlag

	// 生成机器ID作为Slave ID
	var err error
	slaveID, err = slave.GenerateSlaveIDFromMachineID()
	if err != nil {
		fmt.Printf("生成Slave ID失败: %v\n", err)
		os.Exit(1)
	}

	// 设置连接完成回调函数
	slave.SetConnectionCompleteCallback(func(successCount int) {
		log.Printf("所有MQTT客户端连接完成，成功连接数: %d", successCount)
	})

	// 创建配置通道
	configChan := make(chan slave.ConfigData, 10)

	// 启动slave服务器，监听随机端口
	port, messageChan, err := slave.StartSlaveServer(configChan)
	if err != nil {
		fmt.Printf("启动服务器失败: %v\n", err)
		os.Exit(1)
	}

	// 自动注册到master
	fmt.Println("尝试注册到master...")
	err = slave.RegisterToMaster(masterIP, masterPort, slaveID, port)
	if err != nil {
		fmt.Printf("首次注册到master失败: %v\n", err)
		// 不退出程序，继续运行slave服务器
	} else {
		fmt.Println("成功注册到master")
	}

	// 设置Master连接信息供network.go使用
	slave.SetMasterInfo(masterIP, masterPort, slaveID)
	// 设置停止函数供network.go使用
	log.Println("设置停止函数: stopSlaveWithoutStatusChange")
	slave.SetStopFunc(stopSlaveWithoutStatusChange)

	fmt.Printf("Slave ID: %d\n", slaveID)
	fmt.Printf("监听端口: %d\n", port)
	fmt.Printf("Master地址: %s:%d\n", masterIP, masterPort)
	fmt.Printf("pprof地址: http://localhost:%d/debug/pprof/\n", *pprofPortFlag)
	if Version != "" {
		fmt.Printf("版本: %s\n", Version)
	}
	if BuildTime != "" {
		fmt.Printf("构建时间: %s\n", BuildTime)
	}

	// 启动心跳包发送goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Second) // 每5秒发送一次心跳包
		defer ticker.Stop()

		// 失败计数器
		failureCount := 0
		maxFailures := 10 // 失败10次后重新注册

		// 使用for range替代for { select {} }模式
		for range ticker.C {
			err := slave.SendHeartbeat(masterIP, masterPort, slaveID)
			if err != nil {
				failureCount++
				log.Printf("发送心跳包失败 (%d/%d): %v", failureCount, maxFailures, err)
				// 检查是否是因为slave未找到需要重新注册
				if strings.Contains(err.Error(), "need to re-register") || strings.Contains(err.Error(), "Slave not found") || failureCount >= maxFailures {
					log.Println("检测到需要重新注册，正在重新注册到master...")
					err = slave.RegisterToMaster(masterIP, masterPort, slaveID, port)
					if err != nil {
						log.Printf("重新注册到master失败: %v", err)
					} else {
						log.Println("重新注册到master成功")
						// 重置失败计数器
						failureCount = 0
					}
				}
			} else {
				log.Println("心跳包发送成功")
				// 重置失败计数器
				failureCount = 0
			}
		}
	}()

	// 持续监听消息
	log.Println("Slave服务器已启动，等待接收消息...")

	// 在单独的goroutine中处理消息
	go func() {
		for msg := range messageChan {
			log.Printf("处理消息: Type=%s, Content=%v", msg.Type, msg.Content)
			// 这里可以添加处理不同类型消息的逻辑
		}
	}()

	// 在单独的goroutine中处理配置下发
	go func() {
		for config := range configChan {
			log.Printf("处理下发的配置: %+v", config)
			// 实现实际的MQTT连接和订阅逻辑
			// 使用ClientID的值和Start的值开始，到Step结束的循环去连接和订阅
			processConfig(config, masterIP, masterPort, slaveID)

			// 检查是否有启动命令
			if config.Command == "start" {
				log.Printf("收到启动命令，开始连接MQTT服务器")
				// 重置连接计数器
				slave.ResetConnectionCount()

				// 获取最新的配置
				configMutex.RLock()
				if pendingConfig != nil {
					// 在新的goroutine中启动MQTT连接，避免阻塞配置处理
					go func(cfg slave.ConfigData) {
						successCount, failureCount := connectMQTT(cfg)
						message := fmt.Sprintf("MQTT连接完成，成功%d个，失败%d个", successCount, failureCount)
						sendConfigResult(masterIP, masterPort, slaveID, successCount, failureCount, message)
					}(*pendingConfig)
				}
				configMutex.RUnlock()
			}
		}
	}()

	// 主goroutine保持运行
	select {}
}

// 用于存储接收到的配置数据
var pendingConfig *slave.ConfigData
var configMutex = sync.RWMutex{}

// processConfig 处理下发的配置
func processConfig(config slave.ConfigData, masterIP string, masterPort int, slaveID int) {
	log.Printf("开始处理配置: MQTT地址=%s:%d, Topic=%s, QoS=%d, ClientID=%s, Start=%d, Step=%d",
		config.MqttHost, config.MqttPort, config.Topic, config.QoS, config.ClientID, config.Start, config.Step)
	log.Printf("配置数据详情: %+v", config)

	// 计算总客户端数：Step值即为客户端数量
	totalClients := config.Step
	if totalClients <= 0 {
		log.Printf("警告: 配额设置无效，Step(%d) 应该大于 0", config.Step)
		// 发送配置结果反馈给master
		sendConfigResult(masterIP, masterPort, slaveID, 0, 0, "配额设置无效")
		return
	}

	log.Printf("配额信息: 起始值 %d，客户端数量 %d", config.Start, totalClients)

	// 保存配置数据以备后用
	configMutex.Lock()
	pendingConfig = &config
	configMutex.Unlock()

	// 发送配置接收确认给master，表示配置已接收并保存
	message := fmt.Sprintf("配置已接收并保存，共%d个客户端", totalClients)
	sendConfigResult(masterIP, masterPort, slaveID, totalClients, 0, message)
}

// connectMQTT 连接到MQTT服务器
func connectMQTT(config slave.ConfigData) (int, int) {
	// 真实的MQTT连接逻辑
	successCount := 0
	failureCount := 0
	var successMutex sync.Mutex // 用于保护successCount和failureCount的互斥锁

	// 断开所有现有连接
	disconnectAllClients()

	// 设置期望的连接数
	slave.SetExpectedConnections(config.Step)

	// 创建WaitGroup
	var wg sync.WaitGroup

	// 从Start开始创建Step个客户端
	for i := 0; i < config.Step; i++ {
		// 使用符合规范的客户端ID格式：数据库中的client_id + "_" + 7位数字序号
		clientID := fmt.Sprintf("%s_%07d", config.ClientID, config.Start+i)

		// 增加WaitGroup计数器
		wg.Add(1)

		// 为每个客户端启动一个goroutine
		go func(id string, index int) {
			// goroutine结束时减少WaitGroup计数器
			defer wg.Done()

			// 创建MQTT客户端
			mqttClient := slave.NewMQTTClient(config)

			// 连接到MQTT服务器
			err := mqttClient.Connect(id)
			if err != nil {
				log.Printf("创建MQTT客户端 %s 失败: %v", id, err)
				successMutex.Lock()
				failureCount++
				successMutex.Unlock()
				return
			}

			// 存储活跃的客户端
			setActiveClient(id, mqttClient)

			// log.Printf("成功创建并连接MQTT客户端: %s", id)
			successMutex.Lock()
			successCount++
			successMutex.Unlock()
		}(clientID, i)
	}

	// 等待所有goroutine完成
	wg.Wait()

	log.Printf("MQTT连接完成，成功创建 %d 个客户端，失败 %d 个客户端", successCount, failureCount)
	return successCount, failureCount
}

// stopSlaveWithoutStatusChange 停止Slave但不改变状态
func stopSlaveWithoutStatusChange() {
	log.Println("stopSlaveWithoutStatusChange函数被调用")
	// 断开所有MQTT连接
	disconnectAllClients()

	// 发送状态更新到Master，连接数为0，但不改变Slave状态
	sendConfigResultWithoutStatusChange()
}

// disconnectAllClients 断开所有MQTT客户端连接
func disconnectAllClients() {
	log.Println("disconnectAllClients函数被调用，当前活跃连接数:", getActiveClientsCount())

	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for clientID, client := range activeClients {
		log.Printf("断开MQTT客户端 %s 的连接", clientID)
		client.Disconnect()
		delete(activeClients, clientID)
	}

	log.Println("所有MQTT客户端连接已断开")
}

// getActiveClientsCount 获取当前活跃客户端数量
func getActiveClientsCount() int {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	return len(activeClients)
}

// setActiveClient 设置活跃客户端
func setActiveClient(clientID string, client *slave.MQTTClient) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	activeClients[clientID] = client
}

// removeActiveClient 移除活跃客户端
func removeActiveClient(clientID string) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	delete(activeClients, clientID)
}

// getAllActiveClients 获取所有活跃客户端的副本
func getAllActiveClients() map[string]*slave.MQTTClient {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	// 创建副本以避免外部修改
	clientsCopy := make(map[string]*slave.MQTTClient)
	for id, client := range activeClients {
		clientsCopy[id] = client
	}

	return clientsCopy
}

// sendConfigResult 发送配置结果反馈给master
func sendConfigResult(masterIP string, masterPort int, slaveID int, successCount int, failureCount int, message string) {
	// 获取当前活跃连接数
	connections := getActiveClientsCount()

	// 构造配置结果数据
	configResult := ConfigResult{
		SlaveID:      slaveID,
		SuccessCount: successCount,
		FailureCount: failureCount,
		Connections:  connections, // 添加连接数
		Message:      message,
	}

	// 将数据序列化为JSON
	data, err := json.Marshal(configResult)
	if err != nil {
		log.Printf("序列化配置结果失败: %v", err)
		return
	}

	// 构造master的配置结果URL
	configResultURL := fmt.Sprintf("http://%s:%d/config-result", masterIP, masterPort)

	// 创建HTTP客户端，设置超时时间
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发送POST请求
	resp, err := client.Post(configResultURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("发送配置结果到master失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		log.Printf("发送配置结果到master失败，状态码: %d", resp.StatusCode)
		return
	}

	log.Printf("配置结果已发送到master: 成功%d个, 失败%d个, 连接数%d个, 消息: %s", successCount, failureCount, connections, message)
}

// sendConfigResultWithoutStatusChange 发送配置结果反馈给master但不改变状态
func sendConfigResultWithoutStatusChange() {
	// 获取当前活跃连接数（应该为0）
	connections := getActiveClientsCount()

	// 构造配置结果数据
	configResult := ConfigResult{
		SlaveID:      slaveID,
		SuccessCount: 0,
		FailureCount: 0,
		Connections:  connections, // 添加连接数
		Message:      "Slave已停止，所有连接已断开",
	}

	// 将数据序列化为JSON
	data, err := json.Marshal(configResult)
	if err != nil {
		log.Printf("序列化配置结果失败: %v", err)
		return
	}

	// 构造master的配置结果URL
	configResultURL := fmt.Sprintf("http://%s:%d/config-result", masterIP, masterPort)

	// 创建HTTP客户端，设置超时时间
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发送POST请求
	resp, err := client.Post(configResultURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("发送配置结果到master失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		log.Printf("发送配置结果到master失败，状态码: %d", resp.StatusCode)
		return
	}

	log.Printf("配置结果已发送到master: 连接数%d个, 消息: %s", connections, "Slave已停止，所有连接已断开")
}
