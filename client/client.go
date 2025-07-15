// client.go
package client

import (
	"fmt"

	"github.com/friddle/grdp/core"
	"github.com/friddle/grdp/glog"
	"github.com/friddle/grdp/protocol/pdu"
	"github.com/friddle/grdp/protocol/rfb"
)

const (
	CLIP_OFF = 0
	CLIP_IN  = 0x1
	CLIP_OUT = 0x2
)

const (
	TC_RDP = 0
	TC_VNC = 1
)

type Control interface {
	Login(host, user, passwd string, width, height int) error
	KeyUp(sc int, name string)
	KeyDown(sc int, name string)
	MouseMove(x, y int)
	MouseWheel(scroll, x, y int)
	MouseUp(button int, x, y int)
	MouseDown(button int, x, y int)
	On(event string, msg interface{})
	Close()
}

func init() {
	// 配置glog输出到stdout
	glog.InitStdout(glog.INFO)
}

type Client struct {
	host    string
	user    string
	passwd  string
	ctl     Control
	tc      int
	setting *Setting
}

func NewClient(host, user, passwd string, t int, s *Setting) *Client {
	if s == nil {
		s = NewSetting()
	}
	c := &Client{
		host:    host,
		user:    user,
		passwd:  passwd,
		tc:      t,
		setting: s,
	}

	switch t {
	case TC_VNC:
		c.ctl = newVncClient(s)
	default:
		c.ctl = newRdpClient(s)
	}

	s.SetLogLevel()
	return c
}

func (c *Client) Login() error {
	return c.ctl.Login(c.host, c.user, c.passwd, c.setting.Width, c.setting.Height)
}

func (c *Client) KeyUp(sc int, name string) {
	glog.Infof("键盘事件: KeyUp - 扫描码:%d, 按键名称:'%s'", sc, name)
	c.ctl.KeyUp(sc, name)
}
func (c *Client) KeyDown(sc int, name string) {
	glog.Infof("键盘事件: KeyDown - 扫描码:%d, 按键名称:'%s'", sc, name)
	c.ctl.KeyDown(sc, name)
}
func (c *Client) MouseMove(x, y int) {
	glog.Debugf("鼠标事件: MouseMove - 位置(%d, %d)", x, y)
	c.ctl.MouseMove(x, y)
}
func (c *Client) MouseWheel(scroll, x, y int) {
	glog.Infof("鼠标事件: MouseWheel - 滚动:%d, 位置(%d, %d)", scroll, x, y)
	c.ctl.MouseWheel(scroll, x, y)
}
func (c *Client) MouseUp(button, x, y int) {
	buttonName := getMouseButtonName(button)
	glog.Infof("鼠标事件: MouseUp - 按钮:%s(%d), 位置(%d, %d)", buttonName, button, x, y)
	c.ctl.MouseUp(button, x, y)
}
func (c *Client) MouseDown(button, x, y int) {
	buttonName := getMouseButtonName(button)
	glog.Infof("鼠标事件: MouseDown - 按钮:%s(%d), 位置(%d, %d)", buttonName, button, x, y)
	c.ctl.MouseDown(button, x, y)
}
func (c *Client) OnError(f func(e error)) {
	c.ctl.On("error", f)
}
func (c *Client) OnClose(f func()) {
	c.ctl.On("close", f)
}
func (c *Client) OnSuccess(f func()) {
	c.ctl.On("success", f)
}
func (c *Client) OnReady(f func()) {
	c.ctl.On("ready", f)
}
func (c *Client) OnBitmap(f func([]Bitmap)) {
	f1 := func(data interface{}) {
		bs := make([]Bitmap, 0, 50)

		if c.tc == TC_VNC {
			br := data.(*rfb.BitRect)

			for i, v := range br.Rects {

				b := Bitmap{
					DestLeft:     int(v.Rect.X),
					DestTop:      int(v.Rect.Y),
					DestRight:    int(v.Rect.X + v.Rect.Width),
					DestBottom:   int(v.Rect.Y + v.Rect.Height),
					Width:        int(v.Rect.Width),
					Height:       int(v.Rect.Height),
					BitsPerPixel: Bpp(uint16(br.Pf.BitsPerPixel)),
					IsCompress:   false, // VNC数据不压缩
					Data:         v.Data,
				}
				bs = append(bs, b)
			}
		} else {
			bitmapDataList := data.([]pdu.BitmapData)

			for i, v := range bitmapDataList {

				IsCompress := v.IsCompress()
				stream := v.BitmapDataStream
				originalDataSize := len(stream)

				// 无论DecompressOnBackend设置如何，都需要转换为RGBA格式
				// 如果后端解压缩，直接解压缩并转换
				// 如果前端解压缩，也需要先转换为RGBA格式
				if IsCompress && c.setting.DecompressOnBackend {
					stream = bitmapDecompress(&v)
					IsCompress = false
				}

				// 转换为RGBA格式（无论是否压缩）
				rgbaData := convertToRGBA(&v, stream, IsCompress)
				if rgbaData == nil {
					continue
				}

				b := Bitmap{
					DestLeft:     int(v.DestLeft),
					DestTop:      int(v.DestTop),
					DestRight:    int(v.DestRight),
					DestBottom:   int(v.DestBottom),
					Width:        int(v.DestRight - v.DestLeft + 1), // 使用目标显示尺寸
					Height:       int(v.DestBottom - v.DestTop + 1),
					BitsPerPixel: 32,    // RGBA格式固定为32位
					IsCompress:   false, // 前端收到的总是未压缩的RGBA数据
					Data:         rgbaData,
				}
				bs = append(bs, b)
			}
		}

		f(bs)
	}

	c.ctl.On("bitmap", f1)
}

type Bitmap struct {
	DestLeft     int    `json:"destLeft"`
	DestTop      int    `json:"destTop"`
	DestRight    int    `json:"destRight"`
	DestBottom   int    `json:"destBottom"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	BitsPerPixel int    `json:"bitsPerPixel"`
	IsCompress   bool   `json:"isCompress"`
	Data         []byte `json:"data"`
}

func Bpp(bp uint16) int {
	return int(bp / 8)
}

type Setting struct {
	Width               int
	Height              int
	Protocol            string
	LogLevel            glog.LEVEL
	DecompressOnBackend bool // 控制是否在后端解压缩，false表示让前端处理
}

func NewSetting() *Setting {
	return &Setting{
		Width:               1024,
		Height:              768,
		LogLevel:            glog.INFO,
		DecompressOnBackend: false, // 默认前端解压缩
	}
}

func (s *Setting) SetLogLevel() {
	glog.SetLevel(s.LogLevel)
}

// SetDecompressOnBackend 设置是否在后端进行解压缩
func (s *Setting) SetDecompressOnBackend(decompress bool) {
	s.DecompressOnBackend = decompress
	glog.Debugf("设置解压缩策略: 后端解压缩=%v", decompress)
}

func (s *Setting) SetRequestedProtocol(p uint32) {}
func (s *Setting) SetClipboard(c int)            {}

func bitmapDecompress(bitmap *pdu.BitmapData) []byte {
	return core.Decompress(bitmap.BitmapDataStream, int(bitmap.Width), int(bitmap.Height), Bpp(bitmap.BitsPerPixel))
}

// convertToRGBA 将位图数据转换为RGBA格式
func convertToRGBA(bitmap *pdu.BitmapData, data []byte, isCompressed bool) []byte {
	// 计算目标显示尺寸
	targetWidth := int(bitmap.DestRight - bitmap.DestLeft + 1)
	targetHeight := int(bitmap.DestBottom - bitmap.DestTop + 1)

	// 计算期望的RGBA输出大小
	expectedRGBASize := targetWidth * targetHeight * 4

	// 如果数据是压缩的，需要先解压缩
	if isCompressed {
		// 先解压缩为未压缩数据
		decompressedData := core.Decompress(data, int(bitmap.Width), int(bitmap.Height), Bpp(bitmap.BitsPerPixel))
		if len(decompressedData) == 0 {
			return nil
		}
		data = decompressedData
	}

	// 验证原始数据长度
	expectedUncompressedSize := int(bitmap.Width) * int(bitmap.Height) * int(bitmap.BitsPerPixel) / 8
	if len(data) != expectedUncompressedSize {
		glog.Warnf("convertToRGBA: 数据长度不匹配，期望:%d 实际:%d", expectedUncompressedSize, len(data))
		if len(data) < expectedUncompressedSize {
			return nil
		}
		// 如果数据过多，截断到正确长度
		data = data[:expectedUncompressedSize]
	}

	// 创建RGBA输出缓冲区
	rgbaData := make([]byte, expectedRGBASize)

	// 根据位深度进行颜色转换
	switch bitmap.BitsPerPixel {
	case 15:
		convert15ToRGBA(data, rgbaData, int(bitmap.Width), int(bitmap.Height))
	case 16:
		convert16ToRGBA(data, rgbaData, int(bitmap.Width), int(bitmap.Height))
	case 24:
		convert24ToRGBA(data, rgbaData, int(bitmap.Width), int(bitmap.Height))
	case 32:
		convert32ToRGBA(data, rgbaData, int(bitmap.Width), int(bitmap.Height))
	default:
		return nil
	}

	return rgbaData
}

// 颜色转换函数
func convert15ToRGBA(src []byte, dst []byte, width, height int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if i*2+1 >= len(src) || i*4+3 >= len(dst) {
				continue
			}
			// RDP使用小端序，所以低字节在前
			val := uint16(src[i*2]) | uint16(src[i*2+1])<<8
			// 使用与core/io.go中RGB555ToRGB函数相同的转换方式
			r := uint8(val & 0x7C00 >> 7) // 5 bits red -> 8 bits
			g := uint8(val & 0x03E0 >> 2) // 5 bits green -> 8 bits
			b := uint8(val & 0x001F << 3) // 5 bits blue -> 8 bits
			// RGBA顺序
			dst[i*4+0] = r
			dst[i*4+1] = g
			dst[i*4+2] = b
			dst[i*4+3] = 255
		}
	}
}

func convert16ToRGBA(src []byte, dst []byte, width, height int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if i*2+1 >= len(src) || i*4+3 >= len(dst) {
				continue
			}
			// RDP使用小端序，所以低字节在前
			val := uint16(src[i*2]) | uint16(src[i*2+1])<<8
			// 使用与core/io.go中RGB565ToRGB函数相同的转换方式
			r := uint8(val & 0xF800 >> 8) // 5 bits red -> 8 bits
			g := uint8(val & 0x07E0 >> 3) // 6 bits green -> 8 bits
			b := uint8(val & 0x001F << 3) // 5 bits blue -> 8 bits
			// RGBA顺序
			dst[i*4+0] = r
			dst[i*4+1] = g
			dst[i*4+2] = b
			dst[i*4+3] = 255
		}
	}
}

func convert24ToRGBA(src []byte, dst []byte, width, height int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if i*3+2 >= len(src) || i*4+3 >= len(dst) {
				continue
			}
			// RDP使用BGR顺序，需要转换为RGB
			dst[i*4+0] = src[i*3+2] // R (原BGR中的R)
			dst[i*4+1] = src[i*3+1] // G (原BGR中的G)
			dst[i*4+2] = src[i*3+0] // B (原BGR中的B)
			dst[i*4+3] = 255        // A (完全不透明)
		}
	}
}

func convert32ToRGBA(src []byte, dst []byte, width, height int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if i*4+3 >= len(src) || i*4+3 >= len(dst) {
				continue
			}
			// RDP使用BGRA顺序，需要转换为RGBA
			dst[i*4+0] = src[i*4+2] // R (原BGRA中的R)
			dst[i*4+1] = src[i*4+1] // G (原BGRA中的G)
			dst[i*4+2] = src[i*4+0] // B (原BGRA中的B)
			dst[i*4+3] = src[i*4+3] // A (原BGRA中的A)
		}
	}
}

// getMouseButtonName 将鼠标按钮数字转换为可读的名称
func getMouseButtonName(button int) string {
	switch button {
	case 1:
		return "左键"
	case 2:
		return "中键"
	case 3:
		return "右键"
	case 4:
		return "侧键1"
	case 5:
		return "侧键2"
	default:
		return fmt.Sprintf("未知按钮%d", button)
	}
}
