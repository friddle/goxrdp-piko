package client_piko

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	piko_config "github.com/andydunstall/piko/agent/config"
	"github.com/andydunstall/piko/agent/reverseproxy"
	"github.com/andydunstall/piko/client"
	"github.com/andydunstall/piko/pkg/log"
	"github.com/oklog/run"
	"go.uber.org/zap"
)

// ServiceManager 服务管理器
type ServiceManager struct {
	config    *Config
	ctx       context.Context
	cancel    context.CancelFunc
	rdpClient *RdpClient
	webServer *WebServer
	logger    *zap.Logger
}

// NewServiceManager 创建新的服务管理器
func NewServiceManager(config *Config) *ServiceManager {
	ctx, cancel := context.WithCancel(context.Background())

	// 根据环境变量决定日志配置
	buildEnv := os.Getenv("BUILD_ENV")
	var logger *zap.Logger
	var err error

	if buildEnv == "prod" || buildEnv == "production" {
		// 生产环境：使用生产环境配置，只打印 Warning 及以上级别
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
		logger, err = config.Build()
		if err != nil {
			// 如果创建生产环境日志记录器失败，使用 NopLogger
			logger = zap.NewNop()
		}
	} else {
		// 开发环境：使用开发环境配置
		logger, err = zap.NewDevelopment()
		if err != nil {
			// 如果创建开发环境日志记录器失败，使用生产环境配置
			logger, err = zap.NewProduction()
			if err != nil {
				// 如果连生产环境配置也失败，使用 NopLogger（空操作日志记录器）
				logger = zap.NewNop()
			}
		}
	}

	return &ServiceManager{
		config: config,
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}
}

// Start 启动所有服务
func (sm *ServiceManager) Start() error {
	fmt.Printf("🚀 启动 goxrdp-piko 客户端\n")
	fmt.Printf("客户端名称: %s\n", sm.config.Name)
	fmt.Printf("远程服务器: %s\n", sm.config.Remote)
	fmt.Printf("自动退出: %t\n", sm.config.AutoExit)

	// 自动分配可用端口
	sm.config.GoXrdpPort = sm.config.FindAvailablePort()
	fmt.Printf("本地监听端口: %d\n", sm.config.GoXrdpPort)

	// 使用 oklog/run 启动服务
	return sm.startServices()
}

// startServices 使用 oklog/run 启动所有服务
func (sm *ServiceManager) startServices() error {
	var g run.Group

	// 启动 Web 服务器 - 优先启动
	g.Add(func() error {
		sm.webServer = NewWebServer(sm.config, sm.logger)
		// 设置 RDP 客户端引用
		if sm.rdpClient != nil {
			sm.webServer.SetRdpClient(sm.rdpClient)
		}
		return sm.webServer.Start(sm.ctx)
	}, func(error) {
		// Web服务器会在context取消时自动停止
	})

	// 启动 piko 服务
	g.Add(func() error {
		err := sm.startPiko()
		if err != nil {
			sm.logger.Error("启动piko失败", zap.Error(err))
			return err
		}
		// 等待 context 取消
		<-sm.ctx.Done()
		return sm.ctx.Err()
	}, func(error) {
		// piko 服务会在 context 取消时自动停止
	})

	// 信号处理 - 移到主流程中
	g.Add(func() error {
		c := make(chan os.Signal, 1)

		// 根据操作系统设置不同的信号
		if runtime.GOOS == "windows" {
			// Windows 支持 Ctrl+C (SIGINT) 和 Ctrl+Break
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		} else {
			// Unix-like 系统支持更多信号
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		}

		select {
		case sig := <-c:
			sm.logger.Info("收到停止信号，正在关闭服务", zap.String("signal", sig.String()))
			sm.cancel() // 立即取消 context
			return nil
		case <-sm.ctx.Done():
			return sm.ctx.Err()
		}
	}, func(error) {
		sm.cancel()
	})

	// 24小时超时 - 只有当 AutoExit 为 true 时才启用
	if sm.config.AutoExit {
		g.Add(func() error {
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
			defer cancel()

			select {
			case <-timeoutCtx.Done():
				sm.logger.Info("服务运行时间达到24小时，正在停止")
				sm.cancel()
				return nil
			case <-sm.ctx.Done():
				return sm.ctx.Err()
			}
		}, func(error) {
			sm.cancel()
		})
	}

	sm.logger.Info("服务启动成功",
		zap.String("client", sm.config.Name),
		zap.String("remote", sm.config.Remote),
		zap.Int("port", sm.config.GoXrdpPort),
		zap.Bool("autoExit", sm.config.AutoExit))

	fmt.Printf("✅ 服务启动成功！\n")
	if sm.config.Name != "" {
		fmt.Printf("🌐 访问地址: http://localhost:%d/%s\n", sm.config.GoXrdpPort, sm.config.Name)
	} else {
		fmt.Printf("🌐 访问地址: http://localhost:%d\n", sm.config.GoXrdpPort)
	}
	fmt.Printf("按 Ctrl+C 停止服务\n")

	// 运行所有服务
	return g.Run()
}

// Wait 等待服务运行（已废弃，使用 Start 方法）
func (sm *ServiceManager) Wait() {
	fmt.Printf("⚠️  Wait 方法已废弃，请使用 Start 方法\n")
}

// Stop 停止所有服务
func (sm *ServiceManager) Stop() {
	fmt.Printf("🛑 正在停止所有服务...\n")

	// 停止RDP客户端
	if sm.rdpClient != nil {
		fmt.Printf("🔌 正在断开RDP连接...\n")
		sm.rdpClient.Disconnect()
	}

	// 取消context
	sm.cancel()

	fmt.Printf("✅ 服务已停止\n")
}

// startPiko 启动 piko 客户端
func (sm *ServiceManager) startPiko() error {
	// 创建 piko 配置
	fmt.Printf("启动piko中\n")
	remote := sm.config.Remote
	if strings.HasPrefix(remote, "http") {
		remote = sm.config.Remote
	} else {
		remote = fmt.Sprintf("http://%s", sm.config.Remote)
	}
	conf := &piko_config.Config{
		Connect: piko_config.ConnectConfig{
			URL:     remote,
			Timeout: 30 * time.Second,
		},
		Listeners: []piko_config.ListenerConfig{
			{
				EndpointID: sm.config.Name,
				Protocol:   piko_config.ListenerProtocolHTTP,
				Addr:       fmt.Sprintf("127.0.0.1:%d", sm.config.GoXrdpPort),
				AccessLog:  false,
				Timeout:    30 * time.Second,
				TLS:        piko_config.TLSConfig{},
			},
		},
		Log: log.Config{
			Level:      "info",
			Subsystems: []string{},
		},
		GracePeriod: 30 * time.Second,
	}

	// 创建日志记录器
	logger, err := log.NewLogger("info", []string{})
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %v", err)
	}

	// 验证配置
	if err := conf.Validate(); err != nil {
		return fmt.Errorf("piko 配置验证失败: %v", err)
	}

	// 解析连接 URL
	connectURL, err := url.Parse(conf.Connect.URL)
	if err != nil {
		return fmt.Errorf("解析连接 URL 失败: %v", err)
	}

	// 创建上游客户端
	upstream := &client.Upstream{
		URL:       connectURL,
		TLSConfig: nil, // 不使用 TLS
		Logger:    logger.WithSubsystem("client"),
	}

	// 为每个监听器创建连接
	for _, listenerConfig := range conf.Listeners {
		fmt.Printf("正在连接到端点: %s\n", listenerConfig.EndpointID)

		ln, err := upstream.Listen(sm.ctx, listenerConfig.EndpointID)
		if err != nil {
			return fmt.Errorf("监听端点失败 %s: %v", listenerConfig.EndpointID, err)
		}

		fmt.Printf("成功连接到端点: %s\n", listenerConfig.EndpointID)
		// 创建 HTTP 代理服务器，传入正确的配置而不是 nil
		metrics := reverseproxy.NewMetrics("proxy")
		server := reverseproxy.NewServer(listenerConfig, metrics, logger)
		if server == nil {
			return fmt.Errorf("创建 HTTP 代理服务器失败")
		}

		// 启动代理服务器
		go func() {
			if err := server.Serve(ln); err != nil && err != context.Canceled {
				fmt.Printf("代理服务器运行错误: %v\n", err)
			}
		}()
	}

	fmt.Printf("启动piko结束\n")
	return nil
}

// GetLogger 获取日志记录器
func (sm *ServiceManager) GetLogger() *zap.Logger {
	return sm.logger
}
