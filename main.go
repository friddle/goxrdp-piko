package main

import (
	"fmt"
	"os"

	"github.com/friddle/grdp/client_piko"
	"github.com/spf13/cobra"
)

func main() {
	// 在程序启动时检查权限
	client_piko.CheckPermissions()

	rootCmd := MakeMainCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func MakeMainCmd() *cobra.Command {
	var (
		name       string
		remote     string
		serverPort int
		autoExit   bool
		xrdpHost   string
		xrdpPort   int
		xrdpUser   string
		xrdpPass   string
	)

	cmd := &cobra.Command{
		Use:   "goxrdp",
		Short: "goxrdp-piko 客户端 - 基于终端的远程协助工具",
		Long: `goxrdp-piko 是一个基于终端的高效远程协助工具，集成了 goxrdp 和 piko 服务。
专为复杂网络环境下的远程协助而设计，避免传统远程桌面对高带宽的依赖。

使用示例:
  goxrdp --name=my-server --remote=192.168.1.100:8088  # 连接到远程 piko 服务器
  goxrdp --name=client1 --remote=piko.example.com:8022  # 连接到远程 piko 服务器
  goxrdp --name=local --remote=192.168.1.100:8088  # 指定使用 zsh
  goxrdp --name=server --remote=192.168.1.100:8088 --auto-exit=false  # 禁用24小时自动退出
  goxrdp --name=rdp-client --remote=192.168.1.100:8088 --xrdp-host=192.168.1.200 --xrdp-user=admin --xrdp-pass=password  # 同时连接piko和RDP`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 创建配置
			config := &client_piko.Config{
				Name:       name,
				Remote:     remote,
				ServerPort: serverPort,
				AutoExit:   autoExit,
				XrdpHost:   xrdpHost,
				XrdpPort:   xrdpPort,
				XrdpUser:   xrdpUser,
				XrdpPass:   xrdpPass,
			}

			// 如果命令行参数为空，使用自动获取的默认值
			if xrdpHost == "" {
				config.XrdpHost = client_piko.GetLocalIP()
			}
			if xrdpUser == "" {
				config.XrdpUser = client_piko.GetCurrentUser()
			}

			// 验证配置
			if err := config.Validate(); err != nil {
				return fmt.Errorf("配置验证失败: %v", err)
			}

			// 创建服务管理器
			manager := client_piko.NewServiceManager(config)

			// 启动服务（会阻塞直到服务停止）
			if err := manager.Start(); err != nil {
				return fmt.Errorf("启动服务失败: %v", err)
			}

			return nil
		},
	}

	// 添加命令行参数
	cmd.Flags().StringVar(&name, "name", "", "piko 客户端标识名称")
	cmd.Flags().StringVar(&remote, "remote", "", "远程 piko 服务器地址 (格式: host:port)")
	cmd.Flags().IntVar(&serverPort, "server-port", 8022, "piko 服务器端口")
	cmd.Flags().BoolVar(&autoExit, "auto-exit", true, "是否启用24小时自动退出 (默认: true)")

	// RDP相关参数
	cmd.Flags().StringVar(&xrdpHost, "xrdp-host", "", "RDP服务器主机地址")
	cmd.Flags().IntVar(&xrdpPort, "xrdp-port", 3389, "RDP服务器端口 (默认: 3389)")
	cmd.Flags().StringVar(&xrdpUser, "xrdp-user", "", "RDP用户名")
	cmd.Flags().StringVar(&xrdpPass, "xrdp-pass", "", "RDP密码")

	// 设置必需参数
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("remote")

	return cmd
}
