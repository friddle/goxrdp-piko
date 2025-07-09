package client_piko

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

// Config 配置结构体
type Config struct {
	Name       string // piko 客户端名称
	Remote     string // 远程 piko 服务器地址 (格式: host:port)
	ServerPort int    // piko 服务器端口
	XrdpPort   int    // xrdp 端口
	XrdpHost   string // xrdp 主机
	XrdpUser   string // xrdp 用户
	XrdpPass   string // xrdp 密码
	XrdpDomain string // xrdp 域 (为空时使用本地计算机名)
	AutoExit   bool   // 是否启用24小时自动退出 (默认: true)
	GoXrdpPort int    // goxrdp 本地监听端口
}

// NewConfig 创建新的配置实例
func NewConfig() *Config {
	return &Config{
		Name:       getEnvOrDefault("NAME", ""),
		Remote:     getEnvOrDefault("REMOTE", ""),
		ServerPort: getEnvIntOrDefault("SERVER_PORT", 8022),
		AutoExit:   getEnvBoolOrDefault("AUTO_EXIT", true),     // 从环境变量读取自动退出设置，默认为 true
		XrdpHost:   getEnvOrDefault("XRDP_HOST", GetLocalIP()), // 默认获取本机IP
		XrdpPort:   getEnvIntOrDefault("XRDP_PORT", 3389),
		XrdpUser:   getEnvOrDefault("XRDP_USER", GetCurrentUser()), // 默认获取当前用户
		XrdpPass:   getEnvOrDefault("XRDP_PASS", ""),
		XrdpDomain: getEnvOrDefault("XRDP_DOMAIN", GetComputerName()), // 默认获取本地计算机名
		GoXrdpPort: getEnvIntOrDefault("GOXRDP_PORT", 0),              // 0表示自动分配
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("客户端名称不能为空")
	}
	if c.Remote == "" {
		return fmt.Errorf("远程服务器地址不能为空")
	}
	return nil
}

// GetRemoteHost 获取远程主机地址
func (c *Config) GetRemoteHost() string {
	// 解析 remote 参数，格式: host:port
	parts := strings.Split(c.Remote, ":")
	if len(parts) >= 1 {
		return parts[0]
	}
	return "localhost"
}

// GetRemotePort 获取远程端口
func (c *Config) GetRemotePort() int {
	// 解析 remote 参数，格式: host:port
	parts := strings.Split(c.Remote, ":")
	if len(parts) >= 2 {
		if port, err := strconv.Atoi(parts[1]); err == nil {
			return port
		}
	}
	return 8088
}

// FindAvailablePort 查找可用端口，从8080开始
func (c *Config) FindAvailablePort() int {
	startPort := 8080
	for port := startPort; port < startPort+100; port++ {
		if isPortAvailable(port) {
			return port
		}
	}
	return startPort // 如果都不可用，返回默认端口
}

// isPortAvailable 检查端口是否可用
func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// getEnvOrDefault 获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOrDefault 获取整数环境变量或默认值
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBoolOrDefault 获取布尔环境变量或默认值
func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetCurrentUser 获取当前用户名
func GetCurrentUser() string {
	currentUser, err := user.Current()
	if err != nil {
		return "Administrator" // 如果获取失败，返回默认值
	}
	return currentUser.Username
}

// GetLocalIP 获取本机IP地址
func GetLocalIP() string {
	// 尝试获取本机IP地址，使用更安全的方法
	return getLocalIPSafe()
}

// getLocalIPSafe 安全地获取本机IP地址，避免权限问题
func getLocalIPSafe() string {
	// 首先尝试使用简单的网络连接来获取IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// 如果失败，返回默认值
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// getLocalIPFromInterfaces 从网络接口获取IP地址（可能需要权限）
func getLocalIPFromInterfaces() string {
	// 尝试获取本机IP地址
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		// 静默处理权限错误，不显示错误信息
		return "127.0.0.1" // 如果获取失败，返回默认值
	}

	for _, addr := range addrs {
		// 检查是否是IP地址
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			// 只返回IPv4地址
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "127.0.0.1" // 如果没有找到合适的IP，返回默认值
}

// isPermissionError 检查是否是权限错误
func isPermissionError(err error) bool {
	if syscallErr, ok := err.(*os.SyscallError); ok {
		return syscallErr.Err == syscall.EPERM || syscallErr.Err == syscall.EACCES
	}
	return false
}

// CheckPermissions 检查程序运行权限
func CheckPermissions() {
	// 检查是否有root权限
	if os.Geteuid() == 0 {
		fmt.Printf("⚠️  程序以root权限运行，建议使用普通用户权限\n")
	}

	// 静默检查网络权限，不显示错误信息
	_ = canAccessNetwork()
}

// canAccessNetwork 检查是否可以访问网络
func canAccessNetwork() bool {
	// 尝试创建一个简单的网络连接
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 5)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetComputerName 获取计算机名
func GetComputerName() string {
	name, err := os.Hostname()
	if err != nil {
		return "localhost" // 如果获取失败，返回默认值
	}
	return name
}

// GetEffectiveDomain 获取有效的域名称
// 如果配置的域为空，则返回本地计算机名（Windows 10默认行为）
func (c *Config) GetEffectiveDomain() string {
	if c.XrdpDomain == "" {
		return GetComputerName()
	}
	return c.XrdpDomain
}
