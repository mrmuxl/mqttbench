package slave

import (
	"crypto/md5"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// GetMachineID 获取基于CPU信息的机器唯一标识
func GetMachineID() (string, error) {
	var cpuID string
	var err error

	switch runtime.GOOS {
	case "windows":
		cpuID, err = getWindowsCPUID()
	case "linux":
		cpuID, err = getLinuxCPUID()
	case "darwin":
		cpuID, err = getMacCPUID()
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err != nil {
		return "", fmt.Errorf("failed to get CPU ID: %v", err)
	}

	// 使用MD5哈希生成固定长度的标识符
	hash := md5.Sum([]byte(cpuID))
	return fmt.Sprintf("%x", hash), nil
}

// getWindowsCPUID 获取Windows系统的CPU ID
func getWindowsCPUID() (string, error) {
	cmd := exec.Command("wmic", "cpu", "get", "ProcessorId")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute wmic command: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "ProcessorId" {
			return line, nil
		}
	}

	return "", fmt.Errorf("could not find CPU ID in wmic output")
}

// getLinuxCPUID 获取Linux系统的CPU ID
func getLinuxCPUID() (string, error) {
	// 在Linux上，我们可以读取 /proc/cpuinfo
	cmd := exec.Command("cat", "/proc/cpuinfo")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/cpuinfo: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Serial") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	// 如果没有Serial字段，尝试使用machine-id
	cmd = exec.Command("cat", "/etc/machine-id")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read /etc/machine-id: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getMacCPUID 获取macOS系统的CPU ID
func getMacCPUID() (string, error) {
	// 在macOS上，我们可以使用ioreg命令获取硬件UUID
	cmd := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute ioreg command: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				return strings.TrimSpace(strings.Trim(parts[1], "\" ")), nil
			}
		}
	}

	return "", fmt.Errorf("could not find IOPlatformUUID in ioreg output")
}

// GenerateSlaveIDFromMachineID 基于机器ID生成Slave ID
func GenerateSlaveIDFromMachineID() (int, error) {
	machineID, err := GetMachineID()
	if err != nil {
		return 0, fmt.Errorf("failed to get machine ID: %v", err)
	}

	// 取机器ID的前8个字符转换为整数作为Slave ID
	if len(machineID) > 8 {
		machineID = machineID[:8]
	}

	// 将十六进制字符串转换为无符号64位整数，然后取模以适应int范围
	id, err := strconv.ParseUint(machineID, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert machine ID to integer: %v", err)
	}

	// 确保ID在合理范围内（1-1000000）
	slaveID := int(id%999999) + 1

	log.Printf("Generated Slave ID from machine ID: %d", slaveID)
	return slaveID, nil
}
