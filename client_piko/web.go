package client_piko

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"io/fs"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

//go:embed web/*
var webFiles embed.FS

// WebServer Web服务器结构体
type WebServer struct {
	config    *Config
	logger    *zap.Logger
	upgrader  websocket.Upgrader
	clients   map[*websocket.Conn]bool
	broadcast chan interface{}
	rdpClient *RdpClient
	mu        sync.Mutex // 添加互斥锁
	// 刷新操作相关字段
	lastFlushTime time.Time
	flushMutex    sync.Mutex
}

// NewWebServer 创建新的Web服务器
func NewWebServer(config *Config, logger *zap.Logger) *WebServer {
	// 如果传入的 logger 为 nil，创建一个默认的 logger
	if logger == nil {
		var err error
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

	return &WebServer{
		config: config,
		logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan interface{}, 100),
	}
}

// SetRdpClient 设置RDP客户端引用
func (ws *WebServer) SetRdpClient(rdpClient *RdpClient) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.rdpClient = rdpClient
}

// autoConnectRDP 自动连接RDP（如果配置了账号密码）
func (ws *WebServer) autoConnectRDP() {
	// 检查是否配置了必要的RDP连接信息
	if ws.config.XrdpHost == "" || ws.config.XrdpUser == "" || ws.config.XrdpPass == "" {
		ws.logger.Info("RDP自动连接跳过：缺少必要的连接信息",
			zap.String("host", ws.config.XrdpHost),
			zap.String("user", ws.config.XrdpUser),
			zap.Bool("hasPassword", ws.config.XrdpPass != ""))
		return
	}

	ws.logger.Info("检测到RDP配置信息，开始自动连接",
		zap.String("host", ws.config.XrdpHost),
		zap.String("user", ws.config.XrdpUser),
		zap.String("domain", ws.config.XrdpDomain),
		zap.Int("port", ws.config.XrdpPort))

	// 构建RDP主机地址
	rdpHost := ws.config.XrdpHost
	if ws.config.XrdpPort != 0 && ws.config.XrdpPort != 3389 {
		rdpHost = fmt.Sprintf("%s:%d", ws.config.XrdpHost, ws.config.XrdpPort)
	} else {
		rdpHost = fmt.Sprintf("%s:3389", ws.config.XrdpHost)
	}

	// 创建RDP客户端
	newRdpClient := NewRdpClient(
		rdpHost,
		ws.config.XrdpUser,
		ws.config.XrdpPass,
		1280, // 默认宽度
		720,  // 默认高度
		ws,
	)

	// 如果有域名，设置域名
	if ws.config.XrdpDomain != "" {
		newRdpClient.SetDomain(ws.config.XrdpDomain)
	}

	// 设置RDP客户端引用
	ws.mu.Lock()
	ws.rdpClient = newRdpClient
	ws.mu.Unlock()

	ws.logger.Info("RDP客户端创建成功，开始自动连接",
		zap.String("clientHost", newRdpClient.Host),
		zap.String("clientUser", newRdpClient.User))

	// 广播连接状态
	ws.BroadcastStatus(map[string]string{
		"rdp":  "connecting",
		"piko": "connected",
	})

	// 在goroutine中连接RDP
	go func() {
		// 使用带回退机制的连接方法
		err := newRdpClient.ConnectWithFallback()
		if err != nil {
			ws.logger.Error("RDP自动连接失败", zap.Error(err))
			ws.BroadcastLog("error", fmt.Sprintf("RDP自动连接失败: %v", err))
			ws.BroadcastStatus(map[string]string{
				"rdp":  "disconnected",
				"piko": "connected",
			})
			// 连接失败时清理RDP客户端
			ws.logger.Info("自动连接失败，清理RDP客户端")
			ws.mu.Lock()
			ws.rdpClient = nil
			ws.mu.Unlock()
		} else {
			ws.logger.Info("RDP自动连接成功")
			ws.BroadcastLog("success", "RDP自动连接成功")
			ws.BroadcastStatus(map[string]string{
				"rdp":  "connected",
				"piko": "connected",
			})
		}
	}()
}

// Start 启动Web服务器
func (ws *WebServer) Start(ctx context.Context) error {
	// 创建路由器
	router := mux.NewRouter()

	// 设置静态文件服务，使用config.Name作为前缀
	staticPrefix := "/" + ws.config.Name

	// 创建子文件系统，只包含web目录
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		ws.logger.Error("创建web文件系统失败", zap.Error(err))
		return err
	}

	fs := http.FileServer(http.FS(webFS))
	router.HandleFunc(staticPrefix+"/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, staticPrefix+"/html/index.html", http.StatusFound)
	})

	// API路由 - 在基础路由下，必须在静态文件服务之前
	api := router.PathPrefix(staticPrefix + "/html/api").Subrouter()
	api.HandleFunc("/status", ws.handleStatus).Methods("GET")
	api.HandleFunc("/connect", ws.handleConnect).Methods("POST")
	api.HandleFunc("/disconnect", ws.handleDisconnect).Methods("POST")
	api.HandleFunc("/system-info", ws.handleSystemInfo).Methods("GET")
	api.HandleFunc("/rdp-info", ws.handleRDPInfo).Methods("GET")
	api.HandleFunc("/rdp-screen", ws.handleRDPScreen).Methods("GET")
	api.HandleFunc("/test-connection", ws.handleTestConnection).Methods("POST")
	api.HandleFunc("/simple-connect", ws.handleSimpleConnect).Methods("POST")
	api.HandleFunc("/rdp-status", ws.handleRDPStatus).Methods("GET")
	api.HandleFunc("/rdp-reconnect", ws.handleRDPReconnect).Methods("POST")
	api.HandleFunc("/flush", ws.handleFlush).Methods("POST")

	// WebSocket连接处理
	router.HandleFunc(staticPrefix+"/html/ws", ws.handleWebSocket)

	// 默认页面 - 渲染index.html，不跳转（放在静态文件路由之前）
	router.HandleFunc(staticPrefix+"/", ws.handleIndexPage)

	// 根路径的静态文件路由（如果没有前缀，放在带前缀的静态文件路由之前）
	if ws.config.Name == "" {
		// 只处理静态文件路径，排除默认页面
		router.PathPrefix("/css/").Handler(http.StripPrefix("/", fs))
		router.PathPrefix("/js/").Handler(http.StripPrefix("/", fs))
		router.PathPrefix("/img/").Handler(http.StripPrefix("/", fs))
	}

	// 静态文件路由 - 处理web目录下的所有静态资源（放在最后）
	router.PathPrefix(staticPrefix + "/css").Handler(http.StripPrefix(staticPrefix, fs))
	router.PathPrefix(staticPrefix + "/js").Handler(http.StripPrefix(staticPrefix, fs))
	router.PathPrefix(staticPrefix + "/img").Handler(http.StripPrefix(staticPrefix, fs))
	router.PathPrefix(staticPrefix + "/html").Handler(http.StripPrefix(staticPrefix, fs))

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", ws.config.GoXrdpPort),
		Handler: router,
	}

	ws.logger.Info("Web服务器启动",
		zap.String("地址", server.Addr),
		zap.String("静态文件前缀", staticPrefix))

	// 检查是否配置了RDP连接信息，如果配置了则自动连接
	ws.autoConnectRDP()

	// 启动服务器
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ws.logger.Error("Web服务器启动失败", zap.Error(err))
		}
	}()

	// 等待上下文取消
	<-ctx.Done()

	// 优雅关闭服务器
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		ws.logger.Error("Web服务器关闭失败", zap.Error(err))
		return err
	}

	ws.logger.Info("Web服务器已关闭")
	return nil
}

// handleIndexPage 处理默认页面，渲染index.html
func (ws *WebServer) handleIndexPage(w http.ResponseWriter, r *http.Request) {
	// 读取index.html文件
	content, err := webFiles.ReadFile("web/html/index.html")
	if err != nil {
		ws.logger.Error("读取index.html失败", zap.Error(err))
		http.Error(w, "页面加载失败", http.StatusInternalServerError)
		return
	}

	// 如果有前缀，替换资源路径
	if ws.config.Name != "" {
		prefix := "/" + ws.config.Name
		contentStr := string(content)
		// 替换相对路径为带前缀的路径
		contentStr = strings.ReplaceAll(contentStr, `href="./`, fmt.Sprintf(`href="%s/`, prefix))
		contentStr = strings.ReplaceAll(contentStr, `src="./`, fmt.Sprintf(`src="%s/`, prefix))
		// 处理../开头的路径
		contentStr = strings.ReplaceAll(contentStr, `href="../`, fmt.Sprintf(`href="%s/`, prefix))
		contentStr = strings.ReplaceAll(contentStr, `src="../`, fmt.Sprintf(`src="%s/`, prefix))
		content = []byte(contentStr)
	}

	// 设置内容类型
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

// handleWebSocket 处理WebSocket连接
func (ws *WebServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.logger.Error("WebSocket升级失败", zap.Error(err))
		return
	}

	// 添加客户端到连接池
	ws.mu.Lock()
	ws.clients[conn] = true
	ws.mu.Unlock()

	ws.logger.Info("WebSocket客户端已连接", zap.String("地址", conn.RemoteAddr().String()))

	// 处理WebSocket消息
	go ws.handleWebSocketMessages(conn)
}

// handleWebSocketMessages 处理WebSocket消息
func (ws *WebServer) handleWebSocketMessages(conn *websocket.Conn) {
	defer func() {
		// 清理连接
		ws.mu.Lock()
		delete(ws.clients, conn)
		ws.mu.Unlock()
		conn.Close()
		ws.logger.Info("WebSocket客户端已断开", zap.String("地址", conn.RemoteAddr().String()))
	}()

	for {
		// 读取消息
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ws.logger.Error("WebSocket读取消息失败", zap.Error(err))
			}
			break
		}

		// 解析消息
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			ws.logger.Error("解析WebSocket消息失败", zap.Error(err))
			continue
		}

		// 处理不同类型的消息
		ws.handleWebSocketMessage(conn, msg)
	}
}

// handleWebSocketMessage 处理WebSocket消息
func (ws *WebServer) handleWebSocketMessage(conn *websocket.Conn, msg map[string]interface{}) {
	event, ok := msg["event"].(string)
	if !ok {
		ws.logger.Error("WebSocket消息缺少event字段")
		return
	}

	switch event {
	case "infos":
		ws.handleInfosMessage(conn, msg)
	case "mouse":
		ws.handleMouseMessage(conn, msg)
	case "scancode":
		ws.handleScancodeMessage(conn, msg)
	case "wheel":
		ws.handleWheelMessage(conn, msg)
	case "request-initial-bitmap":
		ws.handleRequestInitialBitmap(conn, msg)
	case "flush":
		ws.handleFlushMessage(conn, msg)
	default:
		ws.logger.Warn("未知的WebSocket事件", zap.String("事件", event))
	}
}

// handleInfosMessage 处理连接信息消息
func (ws *WebServer) handleInfosMessage(conn *websocket.Conn, msg map[string]interface{}) {
	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		ws.logger.Error("infos消息缺少data字段")
		return
	}

	ws.logger.Info("收到连接信息",
		zap.String("IP", fmt.Sprintf("%v", data["ip"])),
		zap.Any("端口", data["port"]),
		zap.String("用户名", fmt.Sprintf("%v", data["username"])))

	// 提取连接信息
	ip, _ := data["ip"].(string)
	port, _ := data["port"].(float64)
	domain, _ := data["domain"].(string)
	username, _ := data["username"].(string)
	password, _ := data["password"].(string)

	// 构建RDP主机地址
	rdpHost := ip
	if port != 0 && port != 3389 {
		rdpHost = fmt.Sprintf("%s:%d", ip, int(port))
	} else {
		rdpHost = fmt.Sprintf("%s:3389", ip)
	}

	// 如果用户没有提供用户名，使用配置中的默认值
	if username == "" {
		username = ws.config.XrdpUser
	}

	// 如果用户没有提供密码，使用配置中的默认值
	if password == "" {
		password = ws.config.XrdpPass
	}

	// 如果用户没有提供域名，使用配置中的默认值
	if domain == "" {
		domain = ws.config.XrdpDomain
	}

	// 获取屏幕分辨率
	screen, _ := data["screen"].(map[string]interface{})
	width := 1280
	height := 720
	if screen != nil {
		if w, ok := screen["width"].(float64); ok {
			width = int(w)
		}
		if h, ok := screen["height"].(float64); ok {
			height = int(h)
		}
	}

	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	existingRdpClient := ws.rdpClient
	ws.mu.Unlock()

	// 检查RDP客户端是否已存在且连接正常
	if existingRdpClient != nil {
		if existingRdpClient.IsConnected() {
			// 检查连接信息是否匹配配置
			configHost := ws.config.XrdpHost
			if ws.config.XrdpPort != 0 && ws.config.XrdpPort != 3389 {
				configHost = fmt.Sprintf("%s:%d", ws.config.XrdpHost, ws.config.XrdpPort)
			} else {
				configHost = fmt.Sprintf("%s:3389", ws.config.XrdpHost)
			}

			// 如果请求的连接信息与配置匹配，或者用户没有提供连接信息（使用配置默认值），则复用连接
			if (ip == "" || rdpHost == configHost) &&
				(username == "" || username == ws.config.XrdpUser) &&
				(domain == "" || domain == ws.config.XrdpDomain) {

				ws.logger.Info("RDP连接已存在且配置匹配，复用现有连接",
					zap.String("requestHost", rdpHost),
					zap.String("configHost", configHost),
					zap.String("requestUser", username),
					zap.String("configUser", ws.config.XrdpUser))

				// 直接发送连接成功事件，复用现有连接
				response := map[string]interface{}{
					"event": "rdp-connect",
					"data": map[string]interface{}{
						"reused":  true,
						"message": "复用现有RDP连接",
					},
				}
				responseBytes, _ := json.Marshal(response)
				conn.WriteMessage(websocket.TextMessage, responseBytes)

				// 广播连接状态
				ws.BroadcastStatus(map[string]string{
					"rdp":  "connected",
					"piko": "connected",
				})

				ws.BroadcastLog("info", "新客户端复用现有RDP连接")
				return
			} else {
				ws.logger.Info("RDP连接已存在但配置不匹配，需要新连接",
					zap.String("requestHost", rdpHost),
					zap.String("configHost", configHost),
					zap.String("requestUser", username),
					zap.String("configUser", ws.config.XrdpUser))
			}
		} else {
			// 如果存在RDP客户端但连接已断开，先清理它
			ws.logger.Info("清理已断开的RDP连接")
			existingRdpClient.Disconnect()
			ws.mu.Lock()
			ws.rdpClient = nil
			ws.mu.Unlock()
		}
	}

	ws.logger.Info("创建RDP客户端",
		zap.String("host", rdpHost),
		zap.String("username", username),
		zap.String("domain", domain),
		zap.Int("width", width),
		zap.Int("height", height))

	// 创建RDP客户端
	newRdpClient := NewRdpClient(
		rdpHost,
		username,
		password,
		width,
		height,
		ws,
	)

	// 如果有域名，设置域名
	if domain != "" {
		newRdpClient.SetDomain(domain)
	}

	// 设置RDP客户端引用
	ws.mu.Lock()
	ws.rdpClient = newRdpClient
	ws.mu.Unlock()

	// 广播连接状态
	ws.BroadcastStatus(map[string]string{
		"rdp":  "connecting",
		"piko": "connected",
	})

	// 在goroutine中连接RDP
	go func() {
		// 使用带回退机制的连接方法
		err := newRdpClient.ConnectWithFallback()
		if err != nil {
			ws.logger.Error("RDP连接失败", zap.Error(err))
			// 发送错误事件
			response := map[string]interface{}{
				"event": "rdp-error",
				"data": map[string]interface{}{
					"code":    "CONNECTION_FAILED",
					"message": err.Error(),
				},
			}
			responseBytes, _ := json.Marshal(response)
			conn.WriteMessage(websocket.TextMessage, responseBytes)

			ws.BroadcastLog("error", fmt.Sprintf("RDP连接失败: %v", err))
			ws.BroadcastStatus(map[string]string{
				"rdp":  "disconnected",
				"piko": "connected",
			})

			// 连接失败时清理RDP客户端
			ws.logger.Info("连接失败，清理RDP客户端")
			ws.mu.Lock()
			ws.rdpClient = nil
			ws.mu.Unlock()
		} else {
			ws.logger.Info("RDP连接成功")
			// 发送连接成功事件
			response := map[string]interface{}{
				"event": "rdp-connect",
				"data":  map[string]interface{}{},
			}
			responseBytes, _ := json.Marshal(response)
			conn.WriteMessage(websocket.TextMessage, responseBytes)

			ws.BroadcastLog("success", "RDP连接成功建立")
			ws.BroadcastStatus(map[string]string{
				"rdp":  "connected",
				"piko": "connected",
			})
		}
	}()
}

// handleMouseMessage 处理鼠标消息
func (ws *WebServer) handleMouseMessage(conn *websocket.Conn, msg map[string]interface{}) {
	data, ok := msg["data"].([]interface{})
	if !ok || len(data) < 4 {
		ws.logger.Error("mouse消息格式错误")
		return
	}

	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil || !rdpClient.IsConnected() {
		ws.logger.Debug("RDP客户端未连接，忽略鼠标事件")
		return
	}

	// 提取鼠标事件参数
	x, _ := data[0].(float64)
	y, _ := data[1].(float64)
	button, _ := data[2].(float64)
	pressed, _ := data[3].(bool)

	// 添加详细的调试日志
	ws.logger.Info("收到鼠标事件",
		zap.Float64("x", x),
		zap.Float64("y", y),
		zap.Float64("button", button),
		zap.Bool("pressed", pressed),
		zap.String("buttonName", getButtonName(int(button))))

	// 判断事件类型
	if button == 0 && !pressed {
		// 鼠标移动事件
		rdpClient.MouseMove(int(x), int(y))
		ws.logger.Debug("转发鼠标移动事件到RDP客户端",
			zap.Int("x", int(x)),
			zap.Int("y", int(y)))
	} else {
		// 鼠标按键事件
		ws.logger.Info("转发鼠标按键事件到RDP客户端",
			zap.Int("x", int(x)),
			zap.Int("y", int(y)),
			zap.Int("button", int(button)),
			zap.Bool("pressed", pressed),
			zap.String("buttonName", getButtonName(int(button))))
		rdpClient.SendMouseEvent(int(x), int(y), int(button), pressed)
	}
}

// getButtonName 获取按钮名称用于调试
func getButtonName(button int) string {
	switch button {
	case 0:
		return "左键"
	case 1:
		return "中键"
	case 2:
		return "右键"
	default:
		return fmt.Sprintf("未知按钮%d", button)
	}
}

// handleScancodeMessage 处理键盘扫描码消息
func (ws *WebServer) handleScancodeMessage(conn *websocket.Conn, msg map[string]interface{}) {
	data, ok := msg["data"].([]interface{})
	if !ok || len(data) < 2 {
		ws.logger.Error("scancode消息格式错误")
		return
	}

	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil || !rdpClient.IsConnected() {
		ws.logger.Debug("RDP客户端未连接，忽略键盘事件")
		return
	}

	// 提取键盘事件参数
	scancode, _ := data[0].(float64)
	pressed, _ := data[1].(bool)

	// 发送键盘事件到RDP客户端
	rdpClient.SendKeyboardEvent(int(scancode), pressed)

	ws.logger.Debug("转发键盘事件到RDP客户端",
		zap.Int("scancode", int(scancode)),
		zap.Bool("pressed", pressed))
}

// handleWheelMessage 处理鼠标滚轮消息
func (ws *WebServer) handleWheelMessage(conn *websocket.Conn, msg map[string]interface{}) {
	data, ok := msg["data"].([]interface{})
	if !ok || len(data) < 5 {
		ws.logger.Error("wheel消息格式错误")
		return
	}

	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil || !rdpClient.IsConnected() {
		ws.logger.Debug("RDP客户端未连接，忽略滚轮事件")
		return
	}

	// 提取滚轮事件参数
	x, _ := data[0].(float64)
	y, _ := data[1].(float64)
	step, _ := data[2].(float64)
	positive, _ := data[3].(bool)
	horizontal, _ := data[4].(bool)

	// 发送滚轮事件到RDP客户端
	rdpClient.SendWheelEvent(int(x), int(y), int(step), positive, horizontal)

	ws.logger.Debug("转发滚轮事件到RDP客户端",
		zap.Int("x", int(x)),
		zap.Int("y", int(y)),
		zap.Int("step", int(step)),
		zap.Bool("positive", positive),
		zap.Bool("horizontal", horizontal))
}

// handleRequestInitialBitmap 处理请求重新获取首次页面位图的消息
func (ws *WebServer) handleRequestInitialBitmap(conn *websocket.Conn, msg map[string]interface{}) {
	ws.logger.Info("收到请求重新获取首次页面位图的消息")

	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil || !rdpClient.IsConnected() {
		ws.logger.Warn("RDP客户端未连接，无法获取位图")
		// 发送错误响应
		response := map[string]interface{}{
			"event": "bitmap-error",
			"data": map[string]interface{}{
				"message": "RDP客户端未连接",
			},
		}
		responseBytes, _ := json.Marshal(response)
		conn.WriteMessage(websocket.TextMessage, responseBytes)
		return
	}

	// 请求RDP客户端重新获取首次页面位图
	ws.logger.Info("请求RDP客户端重新获取首次页面位图")

	// 发送响应确认收到请求
	response := map[string]interface{}{
		"event": "bitmap-request-received",
		"data": map[string]interface{}{
			"message": "正在重新获取首次页面位图",
		},
	}
	responseBytes, _ := json.Marshal(response)
	conn.WriteMessage(websocket.TextMessage, responseBytes)

	// 在goroutine中请求重新获取位图，避免阻塞
	go func() {
		// 调用RDP客户端的方法来重新获取首次页面位图
		err := rdpClient.RequestInitialBitmap()
		if err != nil {
			ws.logger.Error("重新获取首次页面位图失败:", zap.Error(err))
			ws.BroadcastLog("error", fmt.Sprintf("重新获取首次页面位图失败: %v", err))
		} else {
			ws.logger.Info("首次页面位图重新获取请求已发送")
			ws.BroadcastLog("info", "首次页面位图重新获取请求已发送")
		}
	}()
}

// handleFlushMessage 处理刷新消息
func (ws *WebServer) handleFlushMessage(conn *websocket.Conn, msg map[string]interface{}) {
	ws.logger.Info("收到WebSocket刷新消息")

	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil || !rdpClient.IsConnected() {
		ws.logger.Warn("RDP客户端未连接，无法执行刷新")
		// 发送错误响应
		response := map[string]interface{}{
			"event": "flush-error",
			"data": map[string]interface{}{
				"message": "RDP客户端未连接",
			},
		}
		responseBytes, _ := json.Marshal(response)
		conn.WriteMessage(websocket.TextMessage, responseBytes)
		return
	}

	// 发送响应确认收到请求
	response := map[string]interface{}{
		"event": "flush-received",
		"data": map[string]interface{}{
			"message": "正在执行刷新操作",
		},
	}
	responseBytes, _ := json.Marshal(response)
	conn.WriteMessage(websocket.TextMessage, responseBytes)

	// 在goroutine中执行刷新操作，避免阻塞
	go func() {
		// 调用RDP客户端的刷新方法
		err := rdpClient.Flush()
		if err != nil {
			ws.logger.Error("刷新操作失败:", zap.Error(err))
			ws.BroadcastLog("error", fmt.Sprintf("刷新操作失败: %v", err))
		} else {
			ws.logger.Info("刷新操作执行成功")
			ws.BroadcastLog("info", "刷新操作执行成功")
		}
	}()
}

// handleFlush 处理HTTP API刷新请求
func (ws *WebServer) handleFlush(w http.ResponseWriter, r *http.Request) {
	ws.logger.Info("收到HTTP API刷新请求")

	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil || !rdpClient.IsConnected() {
		ws.logger.Warn("RDP客户端未连接，无法执行刷新")
		response := map[string]interface{}{
			"success": false,
			"message": "RDP客户端未连接",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 在goroutine中执行刷新操作，避免阻塞HTTP响应
	go func() {
		err := rdpClient.Flush()
		if err != nil {
			ws.logger.Error("刷新操作失败:", zap.Error(err))
			ws.BroadcastLog("error", fmt.Sprintf("刷新操作失败: %v", err))
		} else {
			ws.logger.Info("刷新操作执行成功")
			ws.BroadcastLog("info", "刷新操作执行成功")
		}
	}()

	// 立即返回成功响应
	response := map[string]interface{}{
		"success": true,
		"message": "刷新操作已启动",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// BroadcastMessage 广播消息给所有连接的客户端
func (ws *WebServer) BroadcastMessage(message interface{}) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	messageBytes, err := json.Marshal(message)
	if err != nil {
		ws.logger.Error("序列化广播消息失败", zap.Error(err))
		return
	}

	// 添加调试日志
	if msgMap, ok := message.(map[string]interface{}); ok {
		if event, exists := msgMap["event"].(string); exists {
			ws.logger.Debug("广播消息",
				zap.String("event", event),
				zap.Int("clientCount", len(ws.clients)),
				zap.Int("messageSize", len(messageBytes)))
		}
	}

	for client := range ws.clients {
		err := client.WriteMessage(websocket.TextMessage, messageBytes)
		if err != nil {
			ws.logger.Error("发送广播消息失败", zap.Error(err))
			client.Close()
			delete(ws.clients, client)
		}
	}
}

// BroadcastLog 广播日志消息
func (ws *WebServer) BroadcastLog(level, message string) {
	logMessage := map[string]interface{}{
		"event": "log",
		"data": map[string]interface{}{
			"level":   level,
			"message": message,
			"time":    time.Now().Unix(),
		},
	}
	ws.BroadcastMessage(logMessage)
}

// BroadcastStatus 广播状态更新
func (ws *WebServer) BroadcastStatus(status map[string]string) {
	statusMessage := map[string]interface{}{
		"event": "status",
		"data":  status,
	}
	ws.BroadcastMessage(statusMessage)
}

// BroadcastRDPError 广播RDP错误事件
func (ws *WebServer) BroadcastRDPError(eventType, errorMessage string) {
	// 根据错误类型提供处理建议
	var suggestion string
	switch eventType {
	case "PROTOCOL_NEGOTIATION_FAILED":
		suggestion = "建议尝试禁用SSL/TLS，使用标准RDP协议"
	case "TLS_FAILED":
		suggestion = "TLS连接失败，建议使用标准RDP协议或检查证书"
	case "NLA_FAILED":
		suggestion = "NLA认证失败，请检查用户名密码，或尝试禁用NLA"
	case "CONNECTION_REFUSED":
		suggestion = "连接被拒绝，请检查目标主机是否开启RDP服务(端口3389)"
	case "CONNECTION_TIMEOUT":
		suggestion = "连接超时，请检查网络连接和防火墙设置"
	case "ACCESS_DENIED":
		suggestion = "访问被拒绝，请检查用户权限和认证信息"
	default:
		suggestion = "请检查网络连接和RDP服务配置"
	}

	errorMsg := map[string]interface{}{
		"event": "rdp_error",
		"data": map[string]interface{}{
			"type":       eventType,
			"message":    errorMessage,
			"suggestion": suggestion,
			"time":       time.Now().Unix(),
		},
	}
	ws.BroadcastMessage(errorMsg)

	// 记录详细的错误日志
	ws.logger.Error("RDP错误事件",
		zap.String("type", eventType),
		zap.String("message", errorMessage),
		zap.String("suggestion", suggestion))
}

// BroadcastRDPClose 广播RDP连接关闭事件
func (ws *WebServer) BroadcastRDPClose() {
	closeMessage := map[string]interface{}{
		"event": "rdp_close",
		"data": map[string]interface{}{
			"time": time.Now().Unix(),
		},
	}
	ws.BroadcastMessage(closeMessage)
}

// BroadcastRDPUpdate 广播RDP更新事件
func (ws *WebServer) BroadcastRDPUpdate(updateData map[string]interface{}) {

	updateMessage := map[string]interface{}{
		"event": "rdp-bitmap", // 修改为前端期望的事件名称
		"data":  updateData,
	}
	ws.BroadcastMessage(updateMessage)
}

// handleStatus 处理状态查询
func (ws *WebServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	ws.logger.Info("收到状态查询请求",
		zap.String("remoteAddr", r.RemoteAddr))

	status := map[string]string{
		"rdp":  "disconnected",
		"piko": "connected", // Piko通常保持连接
	}

	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	// 检查RDP连接状态
	if rdpClient != nil {
		ws.logger.Info("RDP客户端存在，检查连接状态",
			zap.String("host", rdpClient.Host),
			zap.String("user", rdpClient.User))

		if rdpClient.IsConnected() {
			ws.logger.Info("RDP客户端连接正常")
			status["rdp"] = "connected"
		} else {
			ws.logger.Warn("RDP客户端存在但未连接")
			status["rdp"] = "disconnected"
		}
	} else {
		ws.logger.Info("RDP客户端不存在")
	}

	ws.logger.Info("返回状态信息", zap.Any("status", status))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleConnect 处理RDP连接请求
func (ws *WebServer) handleConnect(w http.ResponseWriter, r *http.Request) {
	ws.logger.Info("收到RDP连接请求",
		zap.String("remoteAddr", r.RemoteAddr))

	var req struct {
		Host     string `json:"host"`
		Username string `json:"username"`
		Password string `json:"password"`
		Domain   string `json:"domain"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ws.logger.Error("解析连接请求失败", zap.Error(err))
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	existingRdpClient := ws.rdpClient
	ws.mu.Unlock()

	// 检查RDP客户端是否已存在且连接正常
	if existingRdpClient != nil {
		if existingRdpClient.IsConnected() {
			ws.logger.Info("RDP连接已存在，复用现有连接")
			response := map[string]interface{}{
				"success": true,
				"message": "复用现有RDP连接",
				"reused":  true,
				"config": map[string]interface{}{
					"host":     existingRdpClient.Host,
					"username": existingRdpClient.User,
					"password": existingRdpClient.Password, // 添加密码字段
					"domain":   existingRdpClient.Domain,
					"width":    existingRdpClient.Width,
					"height":   existingRdpClient.Height,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

			ws.BroadcastLog("info", "新客户端复用现有RDP连接")
			return
		} else {
			// 如果存在RDP客户端但连接已断开，先清理它
			ws.logger.Info("清理已断开的RDP连接")
			existingRdpClient.Disconnect()
			ws.mu.Lock()
			ws.rdpClient = nil
			ws.mu.Unlock()
		}
	}

	// 构建RDP主机地址，如果用户没有提供host，使用配置中的默认值
	rdpHost := req.Host
	if rdpHost == "" {
		// 使用配置中的默认RDP主机地址
		if ws.config.XrdpPort != 0 && ws.config.XrdpPort != 3389 {
			rdpHost = fmt.Sprintf("%s:%d", ws.config.XrdpHost, ws.config.XrdpPort)
		} else {
			rdpHost = fmt.Sprintf("%s:3389", ws.config.XrdpHost)
		}
	} else if !strings.Contains(rdpHost, ":") {
		rdpHost = fmt.Sprintf("%s:3389", rdpHost)
	}

	// 如果用户没有提供用户名，使用配置中的默认值
	username := req.Username
	if username == "" {
		username = ws.config.XrdpUser
	}

	// 如果用户没有提供密码，使用配置中的默认值
	password := req.Password
	if password == "" {
		password = ws.config.XrdpPass
	}

	// 如果用户没有提供域名，使用配置中的默认值
	domain := req.Domain
	if domain == "" {
		domain = ws.config.XrdpDomain
	}

	// 设置屏幕分辨率，如果用户没有提供，使用默认值
	width := req.Width
	if width == 0 {
		width = 1280
	}

	height := req.Height
	if height == 0 {
		height = 720
	}

	ws.logger.Info("创建新的RDP客户端",
		zap.String("host", rdpHost),
		zap.String("username", username),
		zap.String("domain", domain),
		zap.Int("width", width),
		zap.Int("height", height))

	// 创建RDP客户端 - 确保全局唯一
	newRdpClient := NewRdpClient(
		rdpHost,
		username,
		password,
		width,
		height,
		ws,
	)

	// 如果有域名，设置域名
	if domain != "" {
		newRdpClient.SetDomain(domain)
	}

	// 设置RDP客户端引用
	ws.mu.Lock()
	ws.rdpClient = newRdpClient
	ws.mu.Unlock()

	ws.logger.Info("RDP客户端创建成功，开始连接",
		zap.String("clientHost", newRdpClient.Host),
		zap.String("clientUser", newRdpClient.User))

	// 广播连接状态
	ws.BroadcastStatus(map[string]string{
		"rdp":  "connecting",
		"piko": "connected",
	})

	// 在goroutine中连接RDP
	go func() {
		// 使用带回退机制的连接方法
		err := newRdpClient.ConnectWithFallback()
		if err != nil {
			ws.logger.Error("RDP连接失败", zap.Error(err))
			ws.BroadcastLog("error", fmt.Sprintf("RDP连接失败: %v", err))
			ws.BroadcastStatus(map[string]string{
				"rdp":  "disconnected",
				"piko": "connected",
			})
			// 连接失败时清理RDP客户端
			ws.logger.Info("连接失败，清理RDP客户端")
			ws.mu.Lock()
			ws.rdpClient = nil
			ws.mu.Unlock()
		} else {
			ws.logger.Info("RDP连接成功")
			ws.BroadcastLog("success", "RDP连接成功建立")
			ws.BroadcastStatus(map[string]string{
				"rdp":  "connected",
				"piko": "connected",
			})
		}
	}()

	response := map[string]interface{}{
		"success": true,
		"message": "RDP连接请求已接收，正在连接...",
		"config": map[string]interface{}{
			"host":     rdpHost,
			"username": username,
			"password": password, // 添加密码字段
			"domain":   domain,
			"width":    width,
			"height":   height,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDisconnect 处理RDP断开请求
func (ws *WebServer) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "没有活动的RDP连接",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 断开RDP连接
	rdpClient.Disconnect()

	// 清理RDP客户端引用
	ws.mu.Lock()
	ws.rdpClient = nil
	ws.mu.Unlock()

	response := map[string]interface{}{
		"success": true,
		"message": "RDP连接已断开",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// 广播断开状态
	ws.BroadcastStatus(map[string]string{
		"rdp":  "disconnected",
		"piko": "connected",
	})
}

// handleSystemInfo 处理系统信息查询
func (ws *WebServer) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]string{
		"platform": runtime.GOOS,
		"arch":     runtime.GOARCH,
		"version":  runtime.Version(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleRDPInfo 处理RDP连接信息查询
func (ws *WebServer) handleRDPInfo(w http.ResponseWriter, r *http.Request) {
	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil {
		// 即使没有RDP客户端，也返回配置中的RDP主机地址
		response := map[string]interface{}{
			"connected": false,
			"host":      ws.config.XrdpHost,
			"port":      ws.config.XrdpPort,
			"user":      ws.config.XrdpUser,
			"domain":    ws.config.XrdpDomain,
			"xrdpPass":  ws.config.XrdpPass, // 添加密码字段
			"message":   "没有活动的RDP连接",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	info := rdpClient.GetConnectionInfo()
	// 确保返回的信息包含配置中的端口信息
	if info["port"] == nil {
		info["port"] = ws.config.XrdpPort
	}
	// 添加密码字段
	info["xrdpPass"] = ws.config.XrdpPass
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleRDPScreen 处理RDP屏幕数据请求
func (ws *WebServer) handleRDPScreen(w http.ResponseWriter, r *http.Request) {
	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil {
		http.Error(w, "没有活动的RDP连接", http.StatusNotFound)
		return
	}

	// 这里应该返回屏幕数据
	// 暂时返回简单的状态信息
	response := map[string]interface{}{
		"connected": rdpClient.IsConnected(),
		"width":     rdpClient.Width,
		"height":    rdpClient.Height,
		"message":   "屏幕数据功能待实现",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTestConnection 处理连接测试请求
func (ws *WebServer) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Host     string `json:"host"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 构建RDP主机地址，如果用户没有提供host，使用配置中的默认值
	rdpHost := req.Host
	if rdpHost == "" {
		// 使用配置中的默认RDP主机地址
		if ws.config.XrdpPort != 0 && ws.config.XrdpPort != 3389 {
			rdpHost = fmt.Sprintf("%s:%d", ws.config.XrdpHost, ws.config.XrdpPort)
		} else {
			rdpHost = fmt.Sprintf("%s:3389", ws.config.XrdpHost)
		}
	} else if !strings.Contains(rdpHost, ":") {
		rdpHost = fmt.Sprintf("%s:3389", rdpHost)
	}

	// 如果用户没有提供用户名，使用配置中的默认值
	username := req.Username
	if username == "" {
		username = ws.config.XrdpUser
	}

	// 如果用户没有提供密码，使用配置中的默认值
	password := req.Password
	if password == "" {
		password = ws.config.XrdpPass
	}

	// 创建一个临时的RDP客户端来测试连接
	tempRdpClient := NewRdpClient(
		rdpHost,
		username,
		password,
		1920, // 默认宽度
		1080, // 默认高度
		ws,
	)

	// 尝试连接
	err := tempRdpClient.Connect()
	if err != nil {
		ws.logger.Error("连接测试失败", zap.Error(err))
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("连接测试失败: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 连接成功，立即断开
	tempRdpClient.Disconnect()
	ws.logger.Info("连接测试成功")
	response := map[string]interface{}{
		"success": true,
		"message": "连接测试成功",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSimpleConnect 处理简化连接请求
func (ws *WebServer) handleSimpleConnect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Host     string `json:"host"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 构建RDP主机地址，如果用户没有提供host，使用配置中的默认值
	rdpHost := req.Host
	if rdpHost == "" {
		// 使用配置中的默认RDP主机地址
		if ws.config.XrdpPort != 0 && ws.config.XrdpPort != 3389 {
			rdpHost = fmt.Sprintf("%s:%d", ws.config.XrdpHost, ws.config.XrdpPort)
		} else {
			rdpHost = fmt.Sprintf("%s:3389", ws.config.XrdpHost)
		}
	} else if !strings.Contains(rdpHost, ":") {
		rdpHost = fmt.Sprintf("%s:3389", rdpHost)
	}

	// 如果用户没有提供用户名，使用配置中的默认值
	username := req.Username
	if username == "" {
		username = ws.config.XrdpUser
	}

	// 如果用户没有提供密码，使用配置中的默认值
	password := req.Password
	if password == "" {
		password = ws.config.XrdpPass
	}

	// 创建一个临时的RDP客户端来测试连接
	tempRdpClient := NewRdpClient(
		rdpHost,
		username,
		password,
		1920, // 默认宽度
		1080, // 默认高度
		ws,
	)

	// 尝试连接
	err := tempRdpClient.SimpleConnect()
	if err != nil {
		ws.logger.Error("简化连接失败", zap.Error(err))
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("简化连接失败: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 连接成功，立即断开
	tempRdpClient.Disconnect()
	ws.logger.Info("简化连接成功")
	response := map[string]interface{}{
		"success": true,
		"message": "简化连接成功",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRDPStatus 处理RDP状态查询
func (ws *WebServer) handleRDPStatus(w http.ResponseWriter, r *http.Request) {
	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil {
		response := map[string]interface{}{
			"connected": false,
			"message":   "没有活动的RDP连接",
			"state":     "no_client",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 获取详细的连接信息
	info := rdpClient.GetDetailedConnectionInfo()

	// 确保返回的信息包含配置中的端口信息
	if info["port"] == nil {
		info["port"] = ws.config.XrdpPort
	}

	// 添加配置信息
	info["config"] = map[string]interface{}{
		"xrdpHost":   ws.config.XrdpHost,
		"xrdpPort":   ws.config.XrdpPort,
		"xrdpUser":   ws.config.XrdpUser,
		"xrdpDomain": ws.config.XrdpDomain,
		"xrdpPass":   ws.config.XrdpPass, // 添加密码字段，用于前端判断是否有密码配置
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleRDPReconnect 处理手动重连RDP请求
func (ws *WebServer) handleRDPReconnect(w http.ResponseWriter, r *http.Request) {
	// 使用互斥锁保护RDP客户端访问
	ws.mu.Lock()
	rdpClient := ws.rdpClient
	ws.mu.Unlock()

	if rdpClient == nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "没有活动的RDP连接可以重连",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 广播重连状态
	ws.BroadcastStatus(map[string]string{
		"rdp":  "reconnecting",
		"piko": "connected",
	})

	// 在goroutine中重连RDP
	go func() {
		err := rdpClient.ManualReconnect()
		if err != nil {
			ws.logger.Error("RDP重连失败", zap.Error(err))
			ws.BroadcastLog("error", fmt.Sprintf("RDP重连失败: %v", err))
			ws.BroadcastStatus(map[string]string{
				"rdp":  "disconnected",
				"piko": "connected",
			})
		} else {
			ws.logger.Info("RDP重连成功")
			ws.BroadcastLog("success", "RDP重连成功")
			ws.BroadcastStatus(map[string]string{
				"rdp":  "connected",
				"piko": "connected",
			})
		}
	}()

	response := map[string]interface{}{
		"success": true,
		"message": "RDP重连请求已接收，正在重连...",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
