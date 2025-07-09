package client_piko

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"sync"

	"github.com/friddle/grdp/core"
	"github.com/friddle/grdp/glog"
	"github.com/friddle/grdp/protocol/nla"
	"github.com/friddle/grdp/protocol/pdu"
	"github.com/friddle/grdp/protocol/sec"
	"github.com/friddle/grdp/protocol/t125"
	"github.com/friddle/grdp/protocol/tpkt"
	"github.com/friddle/grdp/protocol/x224"
	"go.uber.org/zap"
)

func init() {
	// 配置glog输出到stdout
	glog.InitStdout(glog.INFO)
}

// RdpClient struct
type RdpClient struct {
	Host      string
	Width     int
	Height    int
	User      string
	Password  string
	Domain    string
	tpkt      *tpkt.TPKT
	x224      *x224.X224
	mcs       *t125.MCSClient
	sec       *sec.Client
	pdu       *pdu.Client
	ctx       context.Context
	cancel    context.CancelFunc
	connected bool
	webServer *WebServer
	// 重连相关字段
	autoReconnect bool
	maxRetries    int
	retryDelay    time.Duration
	reconnectChan chan bool
	// 位图处理器
	bitmapProcessor *BitmapProcessor
	// 刷新操作相关字段
	lastFlushTime time.Time
	flushMutex    sync.Mutex
}

func NewRdpClient(host, user, password string, width, height int, webServer *WebServer) *RdpClient {
	domain, username := splitUser(user)
	ctx, cancel := context.WithCancel(context.Background())

	client := &RdpClient{
		Host:          host,
		Width:         width,
		Height:        height,
		User:          username,
		Password:      password,
		Domain:        domain,
		ctx:           ctx,
		cancel:        cancel,
		webServer:     webServer,
		autoReconnect: true,            // 默认启用自动重连
		maxRetries:    5,               // 最大重试次数
		retryDelay:    3 * time.Second, // 重试延迟
		reconnectChan: make(chan bool, 1),
	}

	// 创建位图处理器，默认在后端解压缩
	client.bitmapProcessor = NewBitmapProcessor(webServer, false)

	return client
}

func splitUser(user string) (domain string, username string) {
	if strings.Index(user, "\\") != -1 {
		t := strings.Split(user, "\\")
		domain = t[0]
		username = t[len(t)-1]
	} else if strings.Index(user, "/") != -1 {
		t := strings.Split(user, "/")
		domain = t[0]
		username = t[len(t)-1]
	} else {
		username = user
	}
	return
}

func (c *RdpClient) Connect() error {
	host := c.Host
	if !strings.Contains(host, ":") {
		host = host + ":3389"
	}

	glog.Info("Connect:", host, "with", c.Domain+"\\"+c.User, ":", c.Password)

	// 尝试解析主机地址
	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		return fmt.Errorf("无效的主机地址: %v", err)
	}

	// 尝试解析IP地址，优先使用IPv4
	var targetIP string
	if ip := net.ParseIP(hostname); ip != nil {
		// 如果是IP地址，优先使用IPv4
		if ip.To4() != nil {
			targetIP = ip.String()
		} else {
			// 如果是IPv6，尝试转换为IPv4映射地址
			targetIP = ip.String()
		}
	} else {
		// 如果是主机名，尝试解析
		ips, err := net.LookupIP(hostname)
		if err != nil {
			return fmt.Errorf("无法解析主机名 %s: %v", hostname, err)
		}

		// 优先选择IPv4地址
		for _, ip := range ips {
			if ip.To4() != nil {
				targetIP = ip.String()
				break
			}
		}

		// 如果没有IPv4地址，使用第一个IPv6地址
		if targetIP == "" && len(ips) > 0 {
			targetIP = ips[0].String()
		}
	}

	if targetIP == "" {
		return fmt.Errorf("无法解析主机地址: %s", hostname)
	}

	// 构建目标地址
	targetAddr := net.JoinHostPort(targetIP, port)
	glog.Info("尝试连接到:", targetAddr)

	// 尝试连接
	conn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		// 如果IPv6连接失败，尝试IPv4
		if strings.Contains(targetIP, ":") {
			glog.Info("IPv6连接失败，尝试IPv4...")
			// 尝试使用localhost的IPv4地址
			conn, err = net.DialTimeout("tcp", "127.0.0.1:"+port, 10*time.Second)
			if err != nil {
				return fmt.Errorf("连接失败 (IPv4): %v", err)
			}
		} else {
			return fmt.Errorf("连接失败: %v", err)
		}
	}

	glog.Info("TCP连接成功，开始RDP握手...")

	// 尝试不同的连接策略
	strategies := []struct {
		name     string
		protocol uint32
		useNLA   bool
	}{
		{"标准RDP协议", x224.PROTOCOL_RDP, false},
		{"SSL协议", x224.PROTOCOL_SSL, false},
		{"NLA协议", x224.PROTOCOL_HYBRID, true},
	}

	for i, strategy := range strategies {
		glog.Info(fmt.Sprintf("尝试策略 %d/%d: %s", i+1, len(strategies), strategy.name))

		// 关闭之前的连接
		if c.tpkt != nil {
			c.tpkt.Close()
		}

		// 重新建立TCP连接
		conn, err = net.DialTimeout("tcp", targetAddr, 10*time.Second)
		if err != nil {
			glog.Error("重新建立TCP连接失败:", err)
			continue
		}

		// 根据策略创建连接
		if strategy.useNLA {
			c.tpkt = tpkt.New(core.NewSocketLayer(conn), nla.NewNTLMv2(c.Domain, c.User, c.Password))
		} else {
			c.tpkt = tpkt.New(core.NewSocketLayer(conn), nil)
		}

		c.x224 = x224.New(c.tpkt)
		c.mcs = t125.NewMCSClient(c.x224)
		c.sec = sec.NewClient(c.mcs)
		c.pdu = pdu.NewClient(c.sec)

		c.mcs.SetClientDesktop(uint16(c.Width), uint16(c.Height))

		c.sec.SetUser(c.User)
		c.sec.SetPwd(c.Password)
		c.sec.SetDomain(c.Domain)

		c.tpkt.SetFastPathListener(c.sec)
		c.sec.SetFastPathListener(c.pdu)
		c.sec.SetChannelSender(c.mcs)

		// 设置事件处理
		connected := make(chan bool, 1)
		connectionError := make(chan error, 1)

		c.pdu.On("error", func(e error) {
			glog.Info("RDP连接错误:", e)

			// 分析错误类型并提供更详细的错误信息
			errorType := "RDP_ERROR"
			errorMessage := e.Error()

			// 根据错误信息分类错误类型
			if strings.Contains(errorMessage, "protocol negotiation failed") {
				errorType = "PROTOCOL_NEGOTIATION_FAILED"
				glog.Info("协议协商失败，可能需要调整安全设置")
			} else if strings.Contains(errorMessage, "TLS start failed") {
				errorType = "TLS_FAILED"
				glog.Info("TLS启动失败，尝试使用标准RDP协议")
			} else if strings.Contains(errorMessage, "NLA start failed") {
				errorType = "NLA_FAILED"
				glog.Info("NLA认证失败，检查用户名密码或尝试禁用NLA")
			} else if strings.Contains(errorMessage, "connection refused") {
				errorType = "CONNECTION_REFUSED"
				glog.Info("连接被拒绝，检查目标主机是否开启RDP服务")
			} else if strings.Contains(errorMessage, "timeout") {
				errorType = "CONNECTION_TIMEOUT"
				glog.Info("连接超时，检查网络连接或防火墙设置")
			} else if strings.Contains(errorMessage, "access denied") {
				errorType = "ACCESS_DENIED"
				glog.Info("访问被拒绝，检查用户权限或认证信息")
			}

			// 广播错误事件
			if c.webServer != nil {
				c.webServer.BroadcastRDPError(errorType, errorMessage)
			}
			connectionError <- e
		}).On("close", func() {
			glog.Info("RDP连接已关闭")
			c.connected = false

			// 广播连接关闭事件
			if c.webServer != nil {
				c.webServer.BroadcastRDPClose()
			}

			// 启动自动重连
			glog.Info("检测到连接关闭，启动自动重连...")
			go c.reconnect()

			connectionError <- fmt.Errorf("connection closed")
		}).On("success", func() {
			glog.Info("RDP连接成功")
			c.connected = true
			connected <- true
		}).On("ready", func() {
			glog.Info("RDP连接就绪")
			c.connected = true
			connected <- true
		}).On("bitmap", func(rectangles []pdu.BitmapData) {
			glog.Debug("Update Bitmap:", len(rectangles))
			c.HandleBitmapUpdate(rectangles)
		})

		// 设置请求的协议
		c.x224.SetRequestedProtocol(strategy.protocol)

		// 尝试连接
		err := c.x224.Connect()
		if err != nil {
			glog.Error(fmt.Sprintf("策略 %s 连接失败: %v", strategy.name, err))
			continue
		}

		// 等待连接结果
		select {
		case <-connected:
			glog.Info(fmt.Sprintf("策略 %s 连接成功！", strategy.name))

			// 连接成功后发送测试bitmap事件
			go func() {
				time.Sleep(2 * time.Second) // 等待2秒后发送测试事件
				glog.Info("发送测试bitmap事件")
				testRectangles := []pdu.BitmapData{
					{
						DestLeft:         0,
						DestTop:          0,
						DestRight:        100,
						DestBottom:       100,
						Width:            100,
						Height:           100,
						BitsPerPixel:     16,
						Flags:            0,
						BitmapLength:     20000,               // 100x100x2 bytes
						BitmapDataStream: make([]byte, 20000), // 创建测试数据
					},
				}
				c.HandleBitmapUpdate(testRectangles)
			}()

			return nil
		case err := <-connectionError:
			glog.Error(fmt.Sprintf("策略 %s 失败: %v", strategy.name, err))
			continue
		case <-time.After(15 * time.Second):
			glog.Error(fmt.Sprintf("策略 %s 超时", strategy.name))
			continue
		}
	}

	// 如果所有策略都失败，尝试简化连接
	glog.Info("所有标准策略都失败，尝试简化连接...")
	return c.SimpleConnect()
}

func (c *RdpClient) Disconnect() {
	if c.tpkt != nil {
		c.tpkt.Close()
	}
	c.connected = false
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *RdpClient) TestConnection() error {
	return c.Connect()
}

func (c *RdpClient) SimpleConnect() error {
	host := c.Host
	if !strings.Contains(host, ":") {
		host = host + ":3389"
	}

	glog.Info("SimpleConnect:", host, "with", c.Domain+"\\"+c.User, ":", c.Password)

	// 尝试解析主机地址
	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		return fmt.Errorf("无效的主机地址: %v", err)
	}

	// 尝试解析IP地址，优先使用IPv4
	var targetIP string
	if ip := net.ParseIP(hostname); ip != nil {
		if ip.To4() != nil {
			targetIP = ip.String()
		} else {
			targetIP = ip.String()
		}
	} else {
		ips, err := net.LookupIP(hostname)
		if err != nil {
			return fmt.Errorf("无法解析主机名 %s: %v", hostname, err)
		}

		for _, ip := range ips {
			if ip.To4() != nil {
				targetIP = ip.String()
				break
			}
		}

		if targetIP == "" && len(ips) > 0 {
			targetIP = ips[0].String()
		}
	}

	if targetIP == "" {
		return fmt.Errorf("无法解析主机地址: %s", hostname)
	}

	targetAddr := net.JoinHostPort(targetIP, port)
	glog.Info("尝试简化连接到:", targetAddr)

	// 尝试建立TCP连接
	conn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		if strings.Contains(targetIP, ":") {
			glog.Info("IPv6连接失败，尝试IPv4...")
			conn, err = net.DialTimeout("tcp", "127.0.0.1:"+port, 10*time.Second)
			if err != nil {
				return fmt.Errorf("连接失败 (IPv4): %v", err)
			}
		} else {
			return fmt.Errorf("连接失败: %v", err)
		}
	}

	glog.Info("TCP连接成功，开始简化RDP握手...")

	// 创建不包含NLA的连接
	c.tpkt = tpkt.New(core.NewSocketLayer(conn), nil)
	c.x224 = x224.New(c.tpkt)
	c.mcs = t125.NewMCSClient(c.x224)
	c.sec = sec.NewClient(c.mcs)
	c.pdu = pdu.NewClient(c.sec)

	c.mcs.SetClientDesktop(uint16(c.Width), uint16(c.Height))

	c.sec.SetUser(c.User)
	c.sec.SetPwd(c.Password)
	c.sec.SetDomain(c.Domain)

	c.tpkt.SetFastPathListener(c.sec)
	c.sec.SetFastPathListener(c.pdu)
	c.sec.SetChannelSender(c.mcs)

	// 设置事件处理
	connected := make(chan bool, 1)
	connectionError := make(chan error, 1)

	c.pdu.On("error", func(e error) {
		glog.Info("on error:", e)
		connectionError <- e
	}).On("close", func() {
		glog.Info("on close")
		c.connected = false

		// 启动自动重连
		glog.Info("检测到连接关闭，启动自动重连...")
		go c.reconnect()

		connectionError <- fmt.Errorf("connection closed")
	}).On("success", func() {
		glog.Info("on success")
		c.connected = true
		connected <- true
	}).On("ready", func() {
		glog.Info("on ready")
		c.connected = true
		connected <- true
	}).On("bitmap", func(rectangles []pdu.BitmapData) {
		glog.Debug("Update Bitmap:", len(rectangles))
		c.HandleBitmapUpdate(rectangles)
	})

	// 尝试标准RDP协议（不使用SSL或NLA）
	c.x224.SetRequestedProtocol(x224.PROTOCOL_RDP)

	err = c.x224.Connect()
	if err != nil {
		return fmt.Errorf("简化RDP连接失败: %v", err)
	}

	// 等待连接结果
	select {
	case <-connected:
		glog.Info("简化RDP连接成功！")

		// 连接成功后发送测试bitmap事件
		go func() {
			time.Sleep(2 * time.Second) // 等待2秒后发送测试事件
			glog.Info("发送测试bitmap事件")
			testRectangles := []pdu.BitmapData{
				{
					DestLeft:         0,
					DestTop:          0,
					DestRight:        100,
					DestBottom:       100,
					Width:            100,
					Height:           100,
					BitsPerPixel:     16,
					Flags:            0,
					BitmapLength:     20000,               // 100x100x2 bytes
					BitmapDataStream: make([]byte, 20000), // 创建测试数据
				},
			}
			c.HandleBitmapUpdate(testRectangles)
		}()

		return nil
	case err := <-connectionError:
		return fmt.Errorf("简化RDP连接失败: %v", err)
	case <-time.After(15 * time.Second):
		return fmt.Errorf("简化RDP连接超时")
	}
}

func (c *RdpClient) IsConnected() bool {
	// 添加详细的连接状态检查
	connected := c.connected

	// 如果标记为已连接，进行更严格的检查
	if connected {
		// 检查底层连接是否真的可用
		if c.tpkt == nil {
			glog.Info("RDP客户端标记为已连接，但tpkt为nil，实际未连接")
			c.connected = false
			return false
		}

		// 检查context是否已取消，但不要立即认为连接断开
		// 因为WebSocket断开可能会影响context，但RDP连接可能仍然有效
		select {
		case <-c.ctx.Done():
			// 只有在context取消且tpkt也为nil时才认为连接断开
			if c.tpkt == nil {
				glog.Info("RDP客户端context已取消且tpkt为nil，确认连接断开")
				c.connected = false
				return false
			}
			// 如果tpkt仍然存在，可能只是WebSocket断开，RDP连接仍然有效
			glog.Debug("RDP客户端context已取消，但tpkt仍然存在，保持连接状态")
		default:
			// context未取消，连接正常
		}
	}

	glog.Debug("RDP连接状态检查",
		zap.Bool("connected", connected),
		zap.String("host", c.Host),
		zap.String("user", c.User))

	return connected
}

func (c *RdpClient) Wait() {
	<-c.ctx.Done()
}

func (c *RdpClient) Close() {
	c.Disconnect()
}

func (c *RdpClient) GetConnectionInfo() map[string]interface{} {
	// 从Host字段中解析主机和端口
	host := c.Host
	port := 3389 // 默认端口

	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		if len(parts) >= 2 {
			host = parts[0]
			if portStr, err := strconv.Atoi(parts[1]); err == nil {
				port = portStr
			}
		}
	}

	return map[string]interface{}{
		"connected": c.connected,
		"host":      host,
		"port":      port,
		"user":      c.User,
		"password":  c.Password, // 添加密码字段
		"domain":    c.Domain,
		"width":     c.Width,
		"height":    c.Height,
	}
}

// GetScreenData 获取RDP屏幕数据
func (c *RdpClient) GetScreenData() (map[string]interface{}, error) {
	if !c.IsConnected() {
		glog.Debug("RDP连接未建立，无法获取屏幕数据")
		return nil, fmt.Errorf("RDP连接未建立")
	}

	// 这里应该返回实际的屏幕数据
	// 目前返回一个占位符数据
	screenData := map[string]interface{}{
		"success":   true,
		"width":     c.Width,
		"height":    c.Height,
		"timestamp": time.Now().Unix(),
		"imageData": nil, // 实际的图像数据将在这里
		"message":   "RDP屏幕数据获取成功",
		"host":      c.Host,
		"user":      c.User,
	}

	return screenData, nil
}

// HandleBitmapUpdate 处理位图更新
func (c *RdpClient) HandleBitmapUpdate(rectangles []pdu.BitmapData) {
	if c.bitmapProcessor != nil {
		c.bitmapProcessor.HandleBitmapUpdate(rectangles)
	} else {
		glog.Warn("位图处理器未初始化，无法处理位图更新")
	}
}

// RequestInitialBitmap 请求重新获取首次页面位图
func (c *RdpClient) RequestInitialBitmap() error {
	if !c.IsConnected() || c.pdu == nil {
		return fmt.Errorf("RDP客户端未连接")
	}

	glog.Info("请求重新获取首次页面位图")

	// 发送一个特殊的请求来触发位图更新
	// 在RDP协议中，我们可以通过发送一个鼠标移动事件来触发屏幕刷新
	// 移动到屏幕中心位置
	centerX := uint16(c.Width / 2)
	centerY := uint16(c.Height / 2)

	p := &pdu.PointerEvent{}
	p.PointerFlags |= pdu.PTRFLAGS_MOVE
	p.XPos = centerX
	p.YPos = centerY

	// 发送鼠标移动事件
	c.pdu.SendInputEvents(pdu.INPUT_EVENT_MOUSE, []pdu.InputEventsInterface{p})

	glog.Info("已发送鼠标移动事件来触发位图更新")

	// 等待一段时间让服务器响应
	time.Sleep(100 * time.Millisecond)

	// 再次发送一个鼠标移动事件，确保触发更新
	p2 := &pdu.PointerEvent{}
	p2.PointerFlags |= pdu.PTRFLAGS_MOVE
	p2.XPos = centerX + 1
	p2.YPos = centerY + 1

	c.pdu.SendInputEvents(pdu.INPUT_EVENT_MOUSE, []pdu.InputEventsInterface{p2})

	glog.Info("已发送第二次鼠标移动事件来触发位图更新")

	return nil
}

// SendMouseEvent 发送鼠标事件
func (c *RdpClient) SendMouseEvent(x, y, button int, pressed bool) {
	if !c.IsConnected() || c.pdu == nil {
		return
	}

	if pressed {
		c.MouseDown(button, x, y)
	} else {
		c.MouseUp(button, x, y)
	}
}

// SendWheelEvent 发送滚轮事件
func (c *RdpClient) SendWheelEvent(x, y, step int, positive, horizontal bool) {
	if !c.IsConnected() || c.pdu == nil {
		return
	}

	// 根据方向调整步数
	if !positive {
		step = -step
	}

	// 发送滚轮事件
	c.MouseWheel(step, x, y)
}

// SendKeyboardEvent 发送键盘事件
func (c *RdpClient) SendKeyboardEvent(scancode int, pressed bool) {
	if !c.IsConnected() || c.pdu == nil {
		return
	}

	if pressed {
		c.KeyDown(scancode, "")
	} else {
		c.KeyUp(scancode, "")
	}
}

// MouseMove 鼠标移动
func (c *RdpClient) MouseMove(x, y int) {
	if !c.IsConnected() || c.pdu == nil {
		return
	}

	p := &pdu.PointerEvent{}
	p.PointerFlags |= pdu.PTRFLAGS_MOVE
	p.XPos = uint16(x)
	p.YPos = uint16(y)
	c.pdu.SendInputEvents(pdu.INPUT_EVENT_MOUSE, []pdu.InputEventsInterface{p})
}

// MouseWheel 鼠标滚轮
func (c *RdpClient) MouseWheel(scroll, x, y int) {
	if !c.IsConnected() || c.pdu == nil {
		return
	}

	p := &pdu.PointerEvent{}
	p.PointerFlags |= pdu.PTRFLAGS_WHEEL
	p.XPos = uint16(x)
	p.YPos = uint16(y)
	// 注意：这里可能需要根据具体的RDP协议实现来调整滚轮事件的处理
	c.pdu.SendInputEvents(pdu.INPUT_EVENT_MOUSE, []pdu.InputEventsInterface{p})
}

// MouseUp 鼠标按键释放
func (c *RdpClient) MouseUp(button int, x, y int) {
	if !c.IsConnected() || c.pdu == nil {
		return
	}

	p := &pdu.PointerEvent{}

	// 添加调试日志
	glog.Info("MouseUp事件",
		zap.Int("button", button),
		zap.Int("x", x),
		zap.Int("y", y),
		zap.String("buttonName", getMouseButtonName(button)))

	switch button {
	case 0:
		p.PointerFlags |= pdu.PTRFLAGS_BUTTON1
		glog.Info("设置PTRFLAGS_BUTTON1 (左键)")
	case 2:
		p.PointerFlags |= pdu.PTRFLAGS_BUTTON2
		glog.Info("设置PTRFLAGS_BUTTON2 (右键)")
	case 1:
		p.PointerFlags |= pdu.PTRFLAGS_BUTTON3
		glog.Info("设置PTRFLAGS_BUTTON3 (中键)")
	default:
		p.PointerFlags |= pdu.PTRFLAGS_MOVE
		glog.Warn("未知按钮，设置为移动事件")
	}

	p.XPos = uint16(x)
	p.YPos = uint16(y)
	c.pdu.SendInputEvents(pdu.INPUT_EVENT_MOUSE, []pdu.InputEventsInterface{p})
}

// MouseDown 鼠标按键按下
func (c *RdpClient) MouseDown(button int, x, y int) {
	if !c.IsConnected() || c.pdu == nil {
		return
	}

	p := &pdu.PointerEvent{}

	p.PointerFlags |= pdu.PTRFLAGS_DOWN

	// 添加调试日志
	glog.Info("MouseDown事件",
		zap.Int("button", button),
		zap.Int("x", x),
		zap.Int("y", y),
		zap.String("buttonName", getMouseButtonName(button)))

	switch button {
	case 0:
		p.PointerFlags |= pdu.PTRFLAGS_BUTTON1
		glog.Info("设置PTRFLAGS_BUTTON1 (左键)")
	case 2:
		p.PointerFlags |= pdu.PTRFLAGS_BUTTON2
		glog.Info("设置PTRFLAGS_BUTTON2 (右键)")
	case 1:
		p.PointerFlags |= pdu.PTRFLAGS_BUTTON3
		glog.Info("设置PTRFLAGS_BUTTON3 (中键)")
	default:
		p.PointerFlags |= pdu.PTRFLAGS_MOVE
		glog.Warn("未知按钮，设置为移动事件")
	}

	p.XPos = uint16(x)
	p.YPos = uint16(y)
	c.pdu.SendInputEvents(pdu.INPUT_EVENT_MOUSE, []pdu.InputEventsInterface{p})
}

// KeyUp 键盘按键释放
func (c *RdpClient) KeyUp(sc int, name string) {
	if !c.IsConnected() || c.pdu == nil {
		return
	}

	p := &pdu.ScancodeKeyEvent{}
	p.KeyCode = uint16(sc)
	p.KeyboardFlags |= pdu.KBDFLAGS_RELEASE
	c.pdu.SendInputEvents(pdu.INPUT_EVENT_SCANCODE, []pdu.InputEventsInterface{p})
}

// KeyDown 键盘按键按下
func (c *RdpClient) KeyDown(sc int, name string) {
	if !c.IsConnected() || c.pdu == nil {
		return
	}

	p := &pdu.ScancodeKeyEvent{}
	p.KeyCode = uint16(sc)
	c.pdu.SendInputEvents(pdu.INPUT_EVENT_SCANCODE, []pdu.InputEventsInterface{p})
}

// reconnect 自动重连方法
func (c *RdpClient) reconnect() {
	if !c.autoReconnect {
		return
	}

	for retryCount := 0; retryCount < c.maxRetries; retryCount++ {
		glog.Info("尝试重连，第", retryCount+1, "次重试...")

		// 等待重试延迟
		select {
		case <-time.After(c.retryDelay):
		case <-c.ctx.Done():
			return
		}

		// 尝试重新连接
		err := c.SimpleConnect()
		if err == nil {
			glog.Info("重连成功！")
			// 通知重连成功
			select {
			case c.reconnectChan <- true:
			default:
			}
			return
		}

		glog.Info("重连失败:", err)
	}

	glog.Info("达到最大重试次数，停止重连")
	// 通知重连失败
	select {
	case c.reconnectChan <- false:
	default:
	}
}

// SetAutoReconnect 设置是否启用自动重连
func (c *RdpClient) SetAutoReconnect(enabled bool) {
	c.autoReconnect = enabled
}

// SetMaxRetries 设置最大重试次数
func (c *RdpClient) SetMaxRetries(maxRetries int) {
	c.maxRetries = maxRetries
}

// SetRetryDelay 设置重试延迟时间
func (c *RdpClient) SetRetryDelay(delay time.Duration) {
	c.retryDelay = delay
}

// GetReconnectStatus 获取重连状态
func (c *RdpClient) GetReconnectStatus() <-chan bool {
	return c.reconnectChan
}

// ManualReconnect 手动触发重连
func (c *RdpClient) ManualReconnect() error {
	glog.Info("手动触发重连...")
	c.connected = false
	if c.tpkt != nil {
		c.tpkt.Close()
	}
	return c.SimpleConnect()
}

// GetReconnectConfig 获取重连配置信息
func (c *RdpClient) GetReconnectConfig() map[string]interface{} {
	return map[string]interface{}{
		"autoReconnect": c.autoReconnect,
		"maxRetries":    c.maxRetries,
		"retryDelay":    c.retryDelay.String(),
		"connected":     c.connected,
	}
}

// SetDecompressOnBackend 设置是否在后端进行解压缩
func (c *RdpClient) SetDecompressOnBackend(decompress bool) {
	if c.bitmapProcessor != nil {
		c.bitmapProcessor.SetDecompressOnBackend(decompress)
	}
}

// GetDecompressOnBackend 获取当前解压缩策略
func (c *RdpClient) GetDecompressOnBackend() bool {
	if c.bitmapProcessor != nil {
		return c.bitmapProcessor.GetDecompressOnBackend()
	}
	return false
}

// SetDomain 设置域名
func (c *RdpClient) SetDomain(domain string) {
	c.Domain = domain
}

// GetDetailedConnectionInfo 获取详细的连接信息
func (c *RdpClient) GetDetailedConnectionInfo() map[string]interface{} {
	// 从Host字段中解析主机和端口
	host := c.Host
	port := 3389 // 默认端口

	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		if len(parts) >= 2 {
			host = parts[0]
			if portStr, err := strconv.Atoi(parts[1]); err == nil {
				port = portStr
			}
		}
	}

	// 检查context状态
	contextCancelled := false
	select {
	case <-c.ctx.Done():
		contextCancelled = true
	default:
		contextCancelled = false
	}

	// 检查tpkt状态
	tpktExists := c.tpkt != nil

	return map[string]interface{}{
		"connected":        c.connected,
		"host":             host,
		"port":             port,
		"user":             c.User,
		"password":         c.Password, // 添加密码字段
		"domain":           c.Domain,
		"width":            c.Width,
		"height":           c.Height,
		"contextCancelled": contextCancelled,
		"tpktExists":       tpktExists,
		"autoReconnect":    c.autoReconnect,
		"maxRetries":       c.maxRetries,
		"retryDelay":       c.retryDelay.String(),
		"connectionState":  c.getConnectionState(),
	}
}

// getConnectionState 获取连接状态描述
func (c *RdpClient) getConnectionState() string {
	if !c.connected {
		return "disconnected"
	}

	// 检查context状态
	select {
	case <-c.ctx.Done():
		if c.tpkt == nil {
			return "disconnected"
		} else {
			return "connected_but_context_cancelled"
		}
	default:
		if c.tpkt == nil {
			return "connected_but_no_tpkt"
		} else {
			return "connected"
		}
	}
}

// ConnectWithFallback 带回退机制的连接方法
func (c *RdpClient) ConnectWithFallback() error {
	// 首先尝试标准连接
	err := c.Connect()
	if err != nil {
		// 如果标准连接失败，检查是否是TLS相关错误
		if strings.Contains(err.Error(), "tls: access denied") ||
			strings.Contains(err.Error(), "NLA start failed") ||
			strings.Contains(err.Error(), "SSL") {
			glog.Info("检测到TLS/NLA错误，尝试禁用安全协议...")
			return c.ConnectWithoutSecurity()
		}
		return err
	}
	return nil
}

// ConnectWithoutSecurity 不使用任何安全协议的连接方法
func (c *RdpClient) ConnectWithoutSecurity() error {
	host := c.Host
	if !strings.Contains(host, ":") {
		host = host + ":3389"
	}

	glog.Info("ConnectWithoutSecurity:", host, "with", c.Domain+"\\"+c.User, ":", c.Password)

	// 尝试解析主机地址
	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		return fmt.Errorf("无效的主机地址: %v", err)
	}

	// 尝试解析IP地址，优先使用IPv4
	var targetIP string
	if ip := net.ParseIP(hostname); ip != nil {
		// 如果是IP地址，优先使用IPv4
		if ip.To4() != nil {
			targetIP = ip.String()
		} else {
			// 如果是IPv6，尝试转换为IPv4映射地址
			targetIP = ip.String()
		}
	} else {
		// 如果是主机名，尝试解析
		ips, err := net.LookupIP(hostname)
		if err != nil {
			return fmt.Errorf("无法解析主机名 %s: %v", hostname, err)
		}

		// 优先选择IPv4地址
		for _, ip := range ips {
			if ip.To4() != nil {
				targetIP = ip.String()
				break
			}
		}

		// 如果没有IPv4地址，使用第一个IPv6地址
		if targetIP == "" && len(ips) > 0 {
			targetIP = ips[0].String()
		}
	}

	if targetIP == "" {
		return fmt.Errorf("无法解析主机地址: %s", hostname)
	}

	// 构建目标地址
	targetAddr := net.JoinHostPort(targetIP, port)
	glog.Info("尝试连接到:", targetAddr)

	// 尝试连接
	conn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		// 如果IPv6连接失败，尝试IPv4
		if strings.Contains(targetIP, ":") {
			glog.Info("IPv6连接失败，尝试IPv4...")
			// 尝试使用localhost的IPv4地址
			conn, err = net.DialTimeout("tcp", "127.0.0.1:"+port, 10*time.Second)
			if err != nil {
				return fmt.Errorf("连接失败 (IPv4): %v", err)
			}
		} else {
			return fmt.Errorf("连接失败: %v", err)
		}
	}

	glog.Info("TCP连接成功，开始RDP握手...")

	// 使用标准RDP协议，不使用NLA
	c.tpkt = tpkt.New(core.NewSocketLayer(conn), nil)
	c.x224 = x224.New(c.tpkt)
	c.mcs = t125.NewMCSClient(c.x224)
	c.sec = sec.NewClient(c.mcs)
	c.pdu = pdu.NewClient(c.sec)

	c.mcs.SetClientDesktop(uint16(c.Width), uint16(c.Height))

	c.sec.SetUser(c.User)
	c.sec.SetPwd(c.Password)
	c.sec.SetDomain(c.Domain)

	// 设置位图更新回调
	c.pdu.On("bitmap", func(rectangles []pdu.BitmapData) {
		c.HandleBitmapUpdate(rectangles)
	})

	// 设置连接关闭回调
	c.pdu.On("close", func() {
		glog.Info("RDP连接已关闭")
		c.connected = false
		if c.webServer != nil {
			c.webServer.BroadcastRDPClose()
		}
	})

	// 设置错误回调
	c.pdu.On("error", func(err error) {
		glog.Error("RDP连接错误:", err)
		c.connected = false

		// 分析错误类型
		errorType := "connection_error"
		errorMessage := err.Error()

		// 根据错误信息分类错误类型
		if strings.Contains(errorMessage, "protocol negotiation failed") {
			errorType = "PROTOCOL_NEGOTIATION_FAILED"
			glog.Info("协议协商失败，可能需要调整安全设置")
		} else if strings.Contains(errorMessage, "TLS start failed") {
			errorType = "TLS_FAILED"
			glog.Info("TLS启动失败，尝试使用标准RDP协议")
		} else if strings.Contains(errorMessage, "NLA start failed") {
			errorType = "NLA_FAILED"
			glog.Info("NLA认证失败，检查用户名密码或尝试禁用NLA")
		} else if strings.Contains(errorMessage, "connection refused") {
			errorType = "CONNECTION_REFUSED"
			glog.Info("连接被拒绝，检查目标主机是否开启RDP服务")
		} else if strings.Contains(errorMessage, "timeout") {
			errorType = "CONNECTION_TIMEOUT"
			glog.Info("连接超时，检查网络连接或防火墙设置")
		} else if strings.Contains(errorMessage, "access denied") {
			errorType = "ACCESS_DENIED"
			glog.Info("访问被拒绝，检查用户权限或认证信息")
		}

		if c.webServer != nil {
			c.webServer.BroadcastRDPError(errorType, errorMessage)
		}
	})

	// 启动连接
	err = c.x224.Connect()
	if err != nil {
		return fmt.Errorf("RDP连接失败: %v", err)
	}

	c.connected = true
	glog.Info("RDP连接成功建立")

	// 启动等待协程
	go c.Wait()

	return nil
}

// Flush 触发RDP协议的界面刷新
func (c *RdpClient) Flush() error {
	if !c.IsConnected() || c.pdu == nil {
		return fmt.Errorf("RDP客户端未连接")
	}

	// 添加刷新频率限制：3秒内只能执行一次
	c.flushMutex.Lock()
	defer c.flushMutex.Unlock()

	now := time.Now()
	if !c.lastFlushTime.IsZero() && now.Sub(c.lastFlushTime) < 3*time.Second {
		glog.Info("刷新操作被限制：距离上次刷新不足3秒")
		return fmt.Errorf("刷新操作过于频繁，请等待3秒后重试")
	}

	c.lastFlushTime = now

	glog.Info("开始执行RDP界面刷新操作")

	// 定义键盘扫描码
	const (
		KEY_ESC       = 0x0001
		KEY_CTRL_LEFT = 0x001D
		KEY_ALT_LEFT  = 0x0038
		KEY_DELETE    = 0xE053
	)

	// 发送Ctrl+Alt+Delete组合键
	glog.Info("发送 Ctrl+Alt+Delete 组合键")

	// 首先按下所有修饰键
	c.KeyDown(KEY_CTRL_LEFT, "ControlLeft")
	time.Sleep(5 * time.Millisecond) // 减少延迟
	c.KeyDown(KEY_ALT_LEFT, "AltLeft")
	time.Sleep(5 * time.Millisecond) // 减少延迟

	// 按下 Delete 键
	c.KeyDown(KEY_DELETE, "Delete")
	time.Sleep(30 * time.Millisecond) // 减少延迟

	// 释放 Delete 键
	c.KeyUp(KEY_DELETE, "Delete")
	time.Sleep(5 * time.Millisecond) // 减少延迟

	// 释放修饰键
	c.KeyUp(KEY_ALT_LEFT, "AltLeft")
	time.Sleep(5 * time.Millisecond) // 减少延迟
	c.KeyUp(KEY_CTRL_LEFT, "ControlLeft")

	glog.Info("Ctrl+Alt+Delete组合键发送完成")

	// 等待一小段时间后发送ESC键
	time.Sleep(200 * time.Millisecond) // 减少延迟

	glog.Info("发送ESC键")
	c.KeyDown(KEY_ESC, "Escape")
	time.Sleep(30 * time.Millisecond) // 减少延迟
	c.KeyUp(KEY_ESC, "Escape")
	time.Sleep(50 * time.Millisecond) // 减少延迟

	glog.Info("RDP界面刷新操作完成")
	return nil
}

// SendABCD 发送ABCD键盘输入
func (c *RdpClient) SendABCD() error {
	if !c.IsConnected() || c.pdu == nil {
		return fmt.Errorf("RDP客户端未连接")
	}

	glog.Info("开始发送ABCD键盘输入")

	// ABCD字符的扫描码定义
	keyCodes := map[string]int{
		"A": 0x001E, // KeyA
		"B": 0x0030, // KeyB
		"C": 0x002E, // KeyC
		"D": 0x0020, // KeyD
	}

	// 依次发送ABCD字符
	for _, char := range []string{"A", "B", "C", "D"} {
		scancode := keyCodes[char]

		// 发送按键按下事件
		c.KeyDown(scancode, char)
		glog.Debug("发送按键按下:", char, "扫描码:", scancode)

		// 等待一小段时间
		time.Sleep(50 * time.Millisecond)

		// 发送按键释放事件
		c.KeyUp(scancode, char)
		glog.Debug("发送按键释放:", char, "扫描码:", scancode)

		// 字符间等待时间
		time.Sleep(100 * time.Millisecond)
	}

	glog.Info("ABCD键盘输入发送完成")
	return nil
}

// getMouseButtonName 获取鼠标按钮名称用于调试
func getMouseButtonName(button int) string {
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
