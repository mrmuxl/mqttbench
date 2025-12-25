package slave

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// RegistrationData 注册数据结构
type RegistrationData struct {
	SlaveID int    `json:"slave_id"`
	IP      string `json:"ip"`
	Port    int    `json:"port"`
}

// RegisterToMaster 向master注册slave信息
func RegisterToMaster(masterIP string, masterPort int, slaveID int, slavePort int) error {
	// 获取本机IP地址
	localIP, err := GetLocalIP()
	if err != nil {
		log.Printf("获取本机IP失败，使用localhost: %v", err)
		localIP = "localhost"
	}

	// 构造注册数据
	registrationData := RegistrationData{
		SlaveID: slaveID,
		IP:      localIP,
		Port:    slavePort,
	}

	// 将数据序列化为JSON
	data, err := json.Marshal(registrationData)
	if err != nil {
		return fmt.Errorf("failed to marshal registration data: %v", err)
	}

	// 构造master的注册URL
	masterAddr := masterIP
	if ip := net.ParseIP(masterIP); ip != nil && ip.To4() == nil {
		// IPv6地址需要用方括号括起来
		masterAddr = fmt.Sprintf("[%s]", masterIP)
	}
	masterURL := fmt.Sprintf("http://%s:%d/register", masterAddr, masterPort)

	log.Printf("发送注册请求到: %s", masterURL)
	log.Printf("注册数据: %s", string(data))

	// 创建HTTP客户端，设置超时时间
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发送POST请求
	resp, err := client.Post(masterURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send registration request: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status code: %d", resp.StatusCode)
	}

	log.Printf("Successfully registered to master at %s:%d", masterIP, masterPort)
	return nil
}

// GetLocalIP 获取本机IP地址
func GetLocalIP() (string, error) {
	// 连接到一个远程地址来确定本机IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// 获取本地地址
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
