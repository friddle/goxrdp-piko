package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ConnectionConfig 连接配置结构
type ConnectionConfig struct {
	IsLocalControl bool   `json:"is_local_control"`
	XrdpHost       string `json:"xrdp_host"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	ConnectionName string `json:"connection_name"`
	RemoteServer   string `json:"remote_server"`
}

// RemoteConfig 远程配置结构
type RemoteConfig struct {
	DefaultRemoteServer string `json:"default_remote_server"`
	DefaultXrdpHost     string `json:"default_xrdp_host"`
	DefaultUsername     string `json:"default_username"`
	DefaultPassword     string `json:"default_password"`
	LastUpdated         int64  `json:"last_updated"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	configFile string
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configFile: ".remote_config.json",
	}
}

// LoadRemoteConfig 加载远程配置
func (cm *ConfigManager) LoadRemoteConfig() *RemoteConfig {
	config := &RemoteConfig{}

	// 检查配置文件是否存在
	if _, err := os.Stat(cm.configFile); os.IsNotExist(err) {
		return config
	}

	// 读取文件内容
	content, err := os.ReadFile(cm.configFile)
	if err != nil {
		fmt.Printf("无法读取配置文件: %v\n", err)
		return config
	}

	// 解析JSON
	if err := json.Unmarshal(content, config); err != nil {
		fmt.Printf("配置文件格式错误: %v\n", err)
		return config
	}

	return config
}

// SaveRemoteConfig 保存远程配置
func (cm *ConfigManager) SaveRemoteConfig(config *RemoteConfig) error {
	// 更新时间戳
	config.LastUpdated = time.Now().Unix()

	// 序列化为JSON
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	// 写入文件
	return os.WriteFile(cm.configFile, content, 0644)
}

// SaveConnectionConfig 保存连接配置
func (cm *ConfigManager) SaveConnectionConfig(config *ConnectionConfig) error {
	remoteConfig := &RemoteConfig{
		DefaultRemoteServer: config.RemoteServer,
		DefaultXrdpHost:     config.XrdpHost,
		DefaultUsername:     config.Username,
		DefaultPassword:     config.Password, // 保存密码到本地
		LastUpdated:         time.Now().Unix(),
	}
	return cm.SaveRemoteConfig(remoteConfig)
}

// ValidateConfig 验证配置
func (cm *ConfigManager) ValidateConfig(config *ConnectionConfig) bool {
	if config.RemoteServer == "" {
		return false
	}

	if config.XrdpHost == "" {
		return false
	}

	if config.Username == "" {
		return false
	}

	if config.Password == "" {
		return false
	}

	if config.ConnectionName == "" {
		config.ConnectionName = config.Username
	}

	return true
}
