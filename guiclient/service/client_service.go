package service

import (
	"fmt"

	"github.com/friddle/grdp/client_piko"
	"github.com/friddle/grdp/guiclient/config"
)

// ClientService 客户端服务
type ClientService struct{}

// NewClientService 创建新的客户端服务
func NewClientService() *ClientService {
	return &ClientService{}
}

// StartClientPiko 启动client_piko服务
func (cs *ClientService) StartClientPiko(config *config.ConnectionConfig) error {
	// 创建client_piko配置
	pikoConfig := &client_piko.Config{
		Name:       config.ConnectionName,
		Remote:     config.RemoteServer,
		XrdpHost:   config.XrdpHost,
		XrdpUser:   config.Username,
		XrdpPass:   config.Password,
		AutoExit:   true,
		GoXrdpPort: 0, // 自动分配端口
	}

	// 验证配置
	if err := pikoConfig.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}

	// 创建服务管理器
	serviceManager := client_piko.NewServiceManager(pikoConfig)

	// 启动服务
	fmt.Printf("正在启动client_piko服务...\n")
	fmt.Printf("连接名称: %s\n", config.ConnectionName)
	fmt.Printf("远程服务器: %s\n", config.RemoteServer)
	fmt.Printf("RDP主机: %s\n", config.XrdpHost)
	fmt.Printf("用户名: %s\n", config.Username)

	// 在goroutine中启动服务，避免阻塞UI
	go func() {
		err := serviceManager.Start()
		if err != nil {
			fmt.Printf("client_piko服务启动失败: %v\n", err)
		}
	}()

	return nil
}

// ShowConnectionInfo 显示连接信息
func (cs *ClientService) ShowConnectionInfo(config *config.ConnectionConfig) {
	fmt.Printf("Connection Info:\n")
	fmt.Printf("  Remote Server: %s\n", config.RemoteServer)
	fmt.Printf("  Connection Name: %s\n", config.ConnectionName)
	fmt.Printf("  XRDP Host: %s\n", config.XrdpHost)
	fmt.Printf("  Username: %s\n", config.Username)
	fmt.Printf("Access URL: https://piko-upstream.friddle.me:8080/%s/html/\n", config.ConnectionName)
}
