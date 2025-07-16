package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"unicode/utf8"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"

	"github.com/friddle/grdp/client_piko"
	"github.com/friddle/grdp/guiclient/config"
	"github.com/friddle/grdp/guiclient/network"
	"github.com/friddle/grdp/guiclient/service"
	"github.com/friddle/grdp/guiclient/ui"
)

type ConnectionConfig struct {
	IsLocalControl bool
	XrdpHost       string
	Username       string
	Password       string
	ConnectionName string
	RemoteServer   string
}

type RemoteConfig struct {
	DefaultRemoteServer string
	DefaultXrdpHost     string
	DefaultUsername     string
}

// 从环境变量获取默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// detectAndDecodeContent 检测并解码文件内容
func detectAndDecodeContent(content []byte) (string, error) {
	// 首先尝试UTF-8
	if utf8.Valid(content) {
		return string(content), nil
	}

	// 如果不是UTF-8，根据操作系统选择合适的编码
	var decoder *encoding.Decoder
	switch runtime.GOOS {
	case "windows":
		// Windows 默认使用 GBK 编码
		decoder = simplifiedchinese.GBK.NewDecoder()
	case "linux", "darwin":
		// Linux 和 macOS 默认使用 UTF-8，但可能包含其他编码
		// 尝试常见的编码
		encodings := []encoding.Encoding{
			simplifiedchinese.GBK,
			simplifiedchinese.GB18030,
			traditionalchinese.Big5,
			japanese.ShiftJIS,
			japanese.EUCJP,
			korean.EUCKR,
		}

		for _, enc := range encodings {
			decoded, _, err := transform.Bytes(enc.NewDecoder(), content)
			if err == nil && utf8.Valid(decoded) {
				return string(decoded), nil
			}
		}
		// 如果都失败了，返回原始内容
		return string(content), nil
	default:
		// 其他系统尝试自动检测编码
		decoder = simplifiedchinese.GBK.NewDecoder()
	}

	if decoder != nil {
		decoded, _, err := transform.Bytes(decoder, content)
		if err != nil {
			return string(content), fmt.Errorf("decode failed: %v", err)
		}
		return string(decoded), nil
	}

	return string(content), nil
}

// saveConfigWithEncoding 使用正确的编码保存配置文件
func saveConfigWithEncoding(config *RemoteConfig, filename string) error {
	var lines []string

	lines = append(lines, "# goxrdp remote connection configuration file")
	lines = append(lines, "# format: key=value")
	lines = append(lines, "# comment lines supported (starting with #)")
	lines = append(lines, "")

	if config.DefaultRemoteServer != "" {
		lines = append(lines, "# default remote server address")
		lines = append(lines, fmt.Sprintf("remote_server=%s", config.DefaultRemoteServer))
		lines = append(lines, "")
	}

	if config.DefaultXrdpHost != "" {
		lines = append(lines, "# default XRDP host address (optional)")
		lines = append(lines, fmt.Sprintf("xrdp_host=%s", config.DefaultXrdpHost))
		lines = append(lines, "")
	}

	if config.DefaultUsername != "" {
		lines = append(lines, "# default username (optional)")
		lines = append(lines, fmt.Sprintf("username=%s", config.DefaultUsername))
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")

	// 根据操作系统选择合适的编码
	var encoder *encoding.Encoder
	switch runtime.GOOS {
	case "windows":
		// Windows 使用 GBK 编码
		encoder = simplifiedchinese.GBK.NewEncoder()
	case "linux", "darwin":
		// Linux 和 macOS 使用 UTF-8 编码
		encoder = nil
	default:
		// 其他系统使用 UTF-8
		encoder = nil
	}

	var encodedContent []byte
	var encodeErr error

	if encoder != nil {
		encodedContent, _, encodeErr = transform.Bytes(encoder, []byte(content))
		if encodeErr != nil {
			return fmt.Errorf("encoding failed: %v", encodeErr)
		}
	} else {
		encodedContent = []byte(content)
	}

	return os.WriteFile(filename, encodedContent, 0644)
}

func loadRemoteConfig() *RemoteConfig {
	config := &RemoteConfig{}

	// 尝试从当前目录读取.remote_config文件
	configFile := ".remote_config"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return config
	}

	// 读取文件内容
	content, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("cannot read config file: %v\n", err)
		return config
	}

	// 检测并解码内容
	contentStr, err := detectAndDecodeContent(content)
	if err != nil {
		fmt.Printf("failed to decode config file: %v\n", err)
		// 如果解码失败，尝试使用原始内容
		contentStr = string(content)
	}

	// 按行解析配置
	lines := strings.Split(contentStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "remote_server":
			config.DefaultRemoteServer = value
		case "xrdp_host":
			config.DefaultXrdpHost = value
		case "username":
			config.DefaultUsername = value
		}
	}

	return config
}

func getCurrentIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func validateConfig(config *ConnectionConfig) bool {
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

// startClientPiko 启动client_piko服务
func startClientPiko(config *ConnectionConfig) error {
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

func showConnectionInfo(config *ConnectionConfig) {
	fmt.Printf("Connection Info:\n")
	fmt.Printf("  Remote Server: %s\n", config.RemoteServer)
	fmt.Printf("  Connection Name: %s\n", config.ConnectionName)
	fmt.Printf("  XRDP Host: %s\n", config.XrdpHost)
	fmt.Printf("  Username: %s\n", config.Username)
	fmt.Printf("Access URL: https://piko-upstream.friddle.me:8080/%s/html/\n", config.ConnectionName)
}

// saveConnectionConfig 保存连接配置到.remote_config文件
func saveConnectionConfig(config *ConnectionConfig) error {
	remoteConfig := &RemoteConfig{
		DefaultRemoteServer: config.RemoteServer,
		DefaultXrdpHost:     config.XrdpHost,
		DefaultUsername:     config.Username,
	}
	return saveConfigWithEncoding(remoteConfig, ".remote_config")
}

func main() {
	// 创建信号通道
	sigChan := make(chan os.Signal, 1)

	// 根据操作系统设置不同的信号
	if runtime.GOOS == "windows" {
		// Windows 支持 Ctrl+C (SIGINT) 和 Ctrl+Break
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	} else {
		// Unix-like 系统支持更多信号
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	}

	// 启动信号处理goroutine
	go func() {
		<-sigChan
		fmt.Println("\n收到退出信号，正在关闭程序...")
		os.Exit(0)
	}()

	go func() {
		w := new(app.Window)
		w.Option(app.Title("goxrdp Remote Connection Tool"))
		w.Option(app.Size(unit.Dp(700), unit.Dp(700)))

		// 创建主题
		th := material.NewTheme()

		// 创建配置管理器
		configManager := config.NewConfigManager()

		// 创建客户端服务
		clientService := service.NewClientService()

		// 创建UI状态
		uiState := ui.NewUIState()

		// 读取配置文件
		remoteConfig := configManager.LoadRemoteConfig()

		// 获取当前IP地址
		currentIP := network.GetCurrentIP()

		// 从环境变量获取默认值
		defaultRemoteServer := network.GetEnvOrDefault("GOXRDP_REMOTE_SERVER", "https://piko-upstream.friddle.me:8082")
		defaultAccessURL := network.GetEnvOrDefault("GOXRDP_ACCESS_URL", "https://piko-upstream.friddle.me:8080")

		// 设置默认值
		if remoteConfig.DefaultRemoteServer != "" {
			uiState.RemoteServerEntry.SetText(remoteConfig.DefaultRemoteServer)
			uiState.Config.RemoteServer = remoteConfig.DefaultRemoteServer
		} else {
			uiState.RemoteServerEntry.SetText(defaultRemoteServer)
			uiState.Config.RemoteServer = defaultRemoteServer
		}

		if remoteConfig.DefaultXrdpHost != "" {
			uiState.HostEntry.SetText(remoteConfig.DefaultXrdpHost)
			uiState.Config.XrdpHost = remoteConfig.DefaultXrdpHost
		}

		if remoteConfig.DefaultUsername != "" {
			uiState.UsernameEntry.SetText(remoteConfig.DefaultUsername)
			uiState.Config.Username = remoteConfig.DefaultUsername
		}

		if remoteConfig.DefaultPassword != "" {
			uiState.PasswordEntry.SetText(remoteConfig.DefaultPassword)
			uiState.Config.Password = remoteConfig.DefaultPassword
		}

		var ops op.Ops

		for {
			e := w.Event()
			switch e := e.(type) {
			case app.DestroyEvent:
				return
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)

				// 处理保存配置按钮点击
				if uiState.SaveConfigBtn.Clicked(gtx) {
					// 更新配置
					uiState.Config.RemoteServer = uiState.RemoteServerEntry.Text()
					uiState.Config.ConnectionName = uiState.ConnectionNameEntry.Text()
					uiState.Config.XrdpHost = uiState.HostEntry.Text()
					uiState.Config.Username = uiState.UsernameEntry.Text()
					uiState.Config.Password = uiState.PasswordEntry.Text()
					uiState.Config.IsLocalControl = uiState.LocalControlCheck.Value

					if configManager.ValidateConfig(uiState.Config) {
						err := configManager.SaveConnectionConfig(uiState.Config)
						if err != nil {
							uiState.StatusText = "配置保存失败: " + err.Error()
							uiState.StatusColor = "red"
							uiState.LastError = err.Error()
						} else {
							uiState.StatusText = "配置已保存到 .remote_config.json 文件"
							uiState.StatusColor = "green"
							uiState.LastError = ""
						}
					} else {
						uiState.StatusText = "配置验证失败，请检查所有必填字段"
						uiState.StatusColor = "red"
						uiState.LastError = "配置验证失败"
					}
				}

				// 处理连接按钮点击
				if uiState.ConnectBtn.Clicked(gtx) {
					// 更新配置
					uiState.Config.RemoteServer = uiState.RemoteServerEntry.Text()
					uiState.Config.ConnectionName = uiState.ConnectionNameEntry.Text()
					uiState.Config.XrdpHost = uiState.HostEntry.Text()
					uiState.Config.Username = uiState.UsernameEntry.Text()
					uiState.Config.Password = uiState.PasswordEntry.Text()
					uiState.Config.IsLocalControl = uiState.LocalControlCheck.Value

					if configManager.ValidateConfig(uiState.Config) {
						uiState.StatusText = "正在连接..."
						uiState.StatusColor = "blue"
						uiState.IsConnected = false
						uiState.LastError = ""

						// 生成访问URL
						uiState.AccessURL = fmt.Sprintf("%s/%s/html/", defaultAccessURL, uiState.Config.ConnectionName)

						// 启动client_piko服务
						err := clientService.StartClientPiko(uiState.Config)
						if err != nil {
							uiState.StatusText = "连接失败: " + err.Error()
							uiState.StatusColor = "red"
							uiState.LastError = err.Error()
							uiState.IsConnected = false
						} else {
							uiState.StatusText = "连接成功！client_piko服务已启动"
							uiState.StatusColor = "green"
							uiState.LastError = ""
							uiState.IsConnected = true
						}
					} else {
						uiState.StatusText = "配置验证失败，请检查所有必填字段"
						uiState.StatusColor = "red"
						uiState.LastError = "配置验证失败"
						uiState.IsConnected = false
					}
				}

				if uiState.QuitBtn.Clicked(gtx) {
					fmt.Println("用户点击退出按钮，正在关闭程序...")
					os.Exit(0)
				}

				// 处理本地控制复选框
				if uiState.LocalControlCheck.Update(gtx) {
					if uiState.LocalControlCheck.Value {
						uiState.Config.XrdpHost = currentIP
						uiState.HostEntry.SetText(currentIP)
					}
				}

				// 处理用户名变化，自动设置连接名称
				currentUsername := uiState.UsernameEntry.Text()
				if currentUsername != uiState.Config.Username {
					uiState.Config.Username = currentUsername
					// 如果连接名称为空，则设置为用户名
					if uiState.ConnectionNameEntry.Text() == "" {
						uiState.ConnectionNameEntry.SetText(currentUsername)
						uiState.Config.ConnectionName = currentUsername
					}
				}

				// 自动保存配置（当所有必填字段都填写完整时，带防抖）
				if uiState.ShouldAutoSave() {
					// 更新配置
					tempConfig := &config.ConnectionConfig{
						RemoteServer:   uiState.RemoteServerEntry.Text(),
						ConnectionName: uiState.ConnectionNameEntry.Text(),
						XrdpHost:       uiState.HostEntry.Text(),
						Username:       uiState.UsernameEntry.Text(),
						Password:       uiState.PasswordEntry.Text(),
						IsLocalControl: uiState.LocalControlCheck.Value,
					}

					if configManager.ValidateConfig(tempConfig) {
						// 静默保存，不显示状态信息
						go func() {
							err := configManager.SaveConnectionConfig(tempConfig)
							if err != nil {
								// 只在控制台输出错误，不影响UI
								fmt.Printf("自动保存配置失败: %v\n", err)
							} else {
								// 更新最后保存时间
								uiState.UpdateLastSaveTime()
							}
						}()
					}
				}

				// 布局
				layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.LayoutTitle(gtx, th)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.LayoutIPInfo(gtx, th, currentIP)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.LayoutFormFields(gtx, th, uiState)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.LayoutButtons(gtx, th, uiState)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.LayoutStatus(gtx, th, uiState)
					}),
				)

				e.Frame(gtx.Ops)
			}
		}
	}()

	app.Main()
}
