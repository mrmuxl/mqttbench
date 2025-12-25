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

// HeartbeatData 心跳包数据结构
type HeartbeatData struct {
	SlaveID   int       `json:"slave_id"`
	Timestamp time.Time `json:"timestamp"`
}

// SendHeartbeat 发送心跳包到master
func SendHeartbeat(masterIP string, masterPort int, slaveID int) error {
	// 构造心跳数据
	heartbeatData := HeartbeatData{
		SlaveID:   slaveID,
		Timestamp: time.Now(),
	}

	// 将数据序列化为JSON
	data, err := json.Marshal(heartbeatData)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat data: %v", err)
	}

	// 构造master的心跳URL
	masterAddr := masterIP
	if ip := net.ParseIP(masterIP); ip != nil && ip.To4() == nil {
		// IPv6地址需要用方括号括起来
		masterAddr = fmt.Sprintf("[%s]", masterIP)
	}
	heartbeatURL := fmt.Sprintf("http://%s:%d/heartbeat", masterAddr, masterPort)

	// 创建HTTP客户端，设置超时时间
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 发送POST请求
	resp, err := client.Post(heartbeatURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send heartbeat request: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		// 如果是404错误，表示slave未找到，需要重新注册
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("slave not found on master, need to re-register: %d", resp.StatusCode)
		}
		return fmt.Errorf("heartbeat failed with status code: %d", resp.StatusCode)
	}

	log.Printf("Heartbeat sent to master at %s:%d", masterIP, masterPort)
	return nil
}
