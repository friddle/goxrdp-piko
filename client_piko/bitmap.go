package client_piko

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/friddle/grdp/glog"
	"github.com/friddle/grdp/protocol/pdu"
)

// BitmapProcessor 位图处理器
type BitmapProcessor struct {
	webServer           *WebServer
	decompressOnBackend bool  // 控制是否在后端解压缩
	lastUpdateTime      int64 // 上次更新时间
	updateInterval      int64 // 更新间隔（毫秒）
}

func GetIsDebug() bool {
	return os.Getenv("IMAGE_DEBUG") == "true"
}

// debugLog 调试日志辅助函数
func debugLog(format string, args ...interface{}) {
	if GetIsDebug() {
		glog.Debugf(format, args...)
	}
}

// debugLogSimple 简单调试日志辅助函数
func debugLogSimple(args ...interface{}) {
	if GetIsDebug() {
		glog.Debug(args...)
	}
}

// NewBitmapProcessor 创建新的位图处理器
func NewBitmapProcessor(webServer *WebServer, decompressOnBackend bool) *BitmapProcessor {
	return &BitmapProcessor{
		webServer:           webServer,
		decompressOnBackend: decompressOnBackend, // 使用传入的参数
		lastUpdateTime:      0,
		updateInterval:      0, // 10ms更新间隔，降低采样率
	}
}

// SetDecompressOnBackend 设置是否在后端进行解压缩
func (bp *BitmapProcessor) SetDecompressOnBackend(decompress bool) {
	bp.decompressOnBackend = decompress
	debugLog("设置解压缩策略: 后端解压缩=%v", decompress)
}

// GetDecompressOnBackend 获取当前解压缩策略
func (bp *BitmapProcessor) GetDecompressOnBackend() bool {
	return bp.decompressOnBackend
}

// SetUpdateInterval 设置位图更新间隔（毫秒）
func (bp *BitmapProcessor) SetUpdateInterval(interval int64) {
	bp.updateInterval = interval
	debugLog("设置位图更新间隔: %dms", interval)
}

// GetUpdateInterval 获取当前位图更新间隔
func (bp *BitmapProcessor) GetUpdateInterval() int64 {
	return bp.updateInterval
}

// HandleBitmapUpdate 处理位图更新
func (bp *BitmapProcessor) HandleBitmapUpdate(rectangles []pdu.BitmapData) {
	// 检查更新间隔，降低采样率
	now := time.Now().UnixMilli()
	if now-bp.lastUpdateTime < bp.updateInterval {
		debugLogSimple("跳过位图更新，间隔太短")
		return
	}
	bp.lastUpdateTime = now

	// debugLogSimple("HandleBitmapUpdate被调用，矩形数量:", len(rectangles))

	if bp.webServer == nil {
		glog.Warn("WebServer未设置，无法处理位图更新")
		return
	}

	// debugLogSimple("处理位图更新，矩形数量:", len(rectangles))

	// 添加详细的矩形信息日志
	for i, rect := range rectangles {
		glog.Debugf("矩形信息 index=%d destLeft=%d destTop=%d destRight=%d destBottom=%d width=%d height=%d bitsPerPixel=%d dataLength=%d",
			i, int(rect.DestLeft), int(rect.DestTop), int(rect.DestRight), int(rect.DestBottom),
			int(rect.Width), int(rect.Height), int(rect.BitsPerPixel), len(rect.BitmapDataStream))
	}

	// 处理位图数据
	var bitmapData []map[string]interface{}

	for i, rect := range rectangles {
		processedData := bp.processRectangle(i, rect)
		if processedData != nil {
			bitmapData = append(bitmapData, processedData)
			glog.Debugf("矩形处理成功 index=%d processedDataSize=%d", i, len(processedData["data"].(string)))
		} else {
			glog.Warnf("矩形处理失败 index=%d", i)
		}
	}

	if len(bitmapData) == 0 {
		glog.Warn("没有有效的位图数据需要发送")
		return
	}

	// 广播位图更新事件
	updateData := map[string]interface{}{
		"bitsPerPixel": rectangles[0].BitsPerPixel, // 使用第一个矩形的位深度
		"rectangles":   bitmapData,
		"timestamp":    time.Now().Unix(),
	}

	// debugLogSimple("准备广播位图更新事件，矩形数量:", len(bitmapData))
	glog.Debugf("位图更新事件详情 rectanglesCount=%d bitsPerPixel=%d timestamp=%d",
		len(bitmapData), int(rectangles[0].BitsPerPixel), time.Now().Unix())

	bp.webServer.BroadcastRDPUpdate(updateData)
	// debugLogSimple("位图更新事件广播完成")
}

// processRectangle 处理单个矩形
func (bp *BitmapProcessor) processRectangle(index int, rect pdu.BitmapData) map[string]interface{} {
	if GetIsDebug() {
		glog.Debug("处理矩形", index, ":", map[string]interface{}{
			"destLeft":     rect.DestLeft,
			"destTop":      rect.DestTop,
			"destRight":    rect.DestRight,
			"destBottom":   rect.DestBottom,
			"width":        rect.Width,
			"height":       rect.Height,
			"bitsPerPixel": rect.BitsPerPixel,
			"isCompress":   !bp.decompressOnBackend,
			"dataLength":   len(rect.BitmapDataStream),
		})
	}

	// 验证数据完整性
	if !bp.validateRectangleData(index, rect) {
		return nil
	}

	// 计算目标矩形尺寸（与前端保持一致）
	targetWidth := int(rect.DestRight - rect.DestLeft + 1)
	targetHeight := int(rect.DestBottom - rect.DestTop + 1)

	// 处理位图数据
	var processedData []byte
	var err error

	if bp.decompressOnBackend {
		// 后端解压缩：直接解压缩并转换为RGBA
		processedData, err = bp.processWithBackendDecompression(index, rect, targetWidth, targetHeight)
	} else {
		// 前端解压缩：也需要转换为RGBA格式，但标记为未压缩
		processedData, err = bp.processWithFrontendDecompression(index, rect, targetWidth, targetHeight)
	}

	if err != nil {
		glog.Error("处理矩形", index, "失败:", err)
		return nil
	}

	// 添加调试信息
	if GetIsDebug() {
		glog.Debug("矩形", index, "处理完成:", map[string]interface{}{
			"processedDataLength": len(processedData),
			"expectedLength":      targetWidth * targetHeight * 4,
			"firstFewBytes": func() []string {
				if len(processedData) >= 16 {
					preview := make([]string, 0)
					for j := 0; j < 16; j++ {
						preview = append(preview, fmt.Sprintf("0x%02x", processedData[j]))
					}
					return preview
				}
				return []string{}
			}(),
		})
	}

	// 发送位图数据 - 与前端格式完全对齐
	return map[string]interface{}{
		"destLeft":     rect.DestLeft,
		"destTop":      rect.DestTop,
		"destRight":    rect.DestRight,
		"destBottom":   rect.DestBottom,
		"width":        uint16(targetWidth),  // 使用目标宽度
		"height":       uint16(targetHeight), // 使用目标高度
		"bitsPerPixel": rect.BitsPerPixel,
		"isCompress":   false, // 无论后端还是前端处理，前端收到的总是未压缩的RGBA数据
		"data":         base64.StdEncoding.EncodeToString(processedData),
	}
}

// validateRectangleData 验证矩形数据
func (bp *BitmapProcessor) validateRectangleData(index int, rect pdu.BitmapData) bool {
	// 验证数据完整性
	if len(rect.BitmapDataStream) == 0 {
		glog.Warn("矩形", index, "数据为空，跳过")
		return false
	}

	if rect.Width <= 0 || rect.Height <= 0 {
		glog.Warn("矩形", index, "尺寸无效:", rect.Width, "x", rect.Height)
		return false
	}

	if rect.DestRight < rect.DestLeft || rect.DestBottom < rect.DestTop {
		glog.Warn("矩形", index, "坐标无效:", rect.DestLeft, rect.DestTop, rect.DestRight, rect.DestBottom)
		return false
	}

	// 验证位深度
	if rect.BitsPerPixel != 15 && rect.BitsPerPixel != 16 && rect.BitsPerPixel != 24 && rect.BitsPerPixel != 32 {
		glog.Warn("矩形", index, "不支持的位深度:", rect.BitsPerPixel)
		return false
	}

	// 计算目标矩形尺寸（与前端保持一致）
	targetWidth := int(rect.DestRight - rect.DestLeft + 1)
	targetHeight := int(rect.DestBottom - rect.DestTop + 1)

	// 验证目标尺寸
	if targetWidth <= 0 || targetHeight <= 0 {
		glog.Warn("矩形", index, "目标尺寸无效:", targetWidth, "x", targetHeight)
		return false
	}

	// 新增：检测全黑数据
	bp.detectBlackBitmap(index, rect)

	return true
}

// detectBlackBitmap 检测全黑位图数据
func (bp *BitmapProcessor) detectBlackBitmap(index int, rect pdu.BitmapData) {
	if len(rect.BitmapDataStream) == 0 {
		return
	}

	// 检查数据是否全为0
	allZero := true
	sampleCount := 0
	nonZeroCount := 0
	maxSample := min(100, len(rect.BitmapDataStream)) // 最多检查100个字节

	for i := 0; i < maxSample; i++ {
		if rect.BitmapDataStream[i] != 0 {
			nonZeroCount++
			// 如果发现非零字节，检查是否足够多
			if nonZeroCount > 5 { // 允许最多5个非零字节
				allZero = false
				break
			}
		}
		sampleCount++
	}

	// 修复：添加更严格的检测条件
	if allZero && len(rect.BitmapDataStream) > 10 {
		glog.Warn("⚠️ 检测到全黑位图数据！矩形", index, "详情:", map[string]interface{}{
			"destLeft":     rect.DestLeft,
			"destTop":      rect.DestTop,
			"destRight":    rect.DestRight,
			"destBottom":   rect.DestBottom,
			"width":        rect.Width,
			"height":       rect.Height,
			"bitsPerPixel": rect.BitsPerPixel,
			"isCompress":   rect.IsCompress(),
			"dataLength":   len(rect.BitmapDataStream),
			"sampleCount":  sampleCount,
			"nonZeroCount": nonZeroCount,
		})

		// 显示前20个字节的详细信息
		previewBytes := make([]string, 0)
		for j := 0; j < min(20, len(rect.BitmapDataStream)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", rect.BitmapDataStream[j]))
		}
		glog.Warn("矩形", index, "全黑数据预览:", strings.Join(previewBytes, ", "))

		// 修复：对于全0数据，建议跳过处理或使用默认颜色
		glog.Warn("矩形", index, "建议：跳过全0数据或使用默认颜色填充")
	}
}

// processWithBackendDecompression 在后端进行解压缩处理
func (bp *BitmapProcessor) processWithBackendDecompression(index int, rect pdu.BitmapData, targetWidth, targetHeight int) ([]byte, error) {
	// 计算期望的未压缩数据大小（使用原始数据尺寸）
	expectedUncompressedSize := int(rect.Width) * int(rect.Height) * int(rect.BitsPerPixel) / 8

	// 计算期望的RGBA输出大小（使用目标显示尺寸）
	expectedRGBASize := targetWidth * targetHeight * 4

	if GetIsDebug() {
		glog.Debug("矩形", index, "尺寸信息:", map[string]interface{}{
			"originalWidth":            rect.Width,
			"originalHeight":           rect.Height,
			"targetWidth":              targetWidth,
			"targetHeight":             targetHeight,
			"bitsPerPixel":             rect.BitsPerPixel,
			"expectedUncompressedSize": expectedUncompressedSize,
			"expectedRGBASize":         expectedRGBASize,
			"actualDataSize":           len(rect.BitmapDataStream),
		})
	}

	// 处理压缩数据 - 在后端解压缩
	var decompressedData []byte

	if rect.IsCompress() {
		decompressedData = bp.decompressBitmapData(index, rect, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize)
	} else {
		decompressedData = bp.convertUncompressedData(index, rect, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize)
	}

	// 验证最终数据
	if len(decompressedData) == 0 {
		return nil, fmt.Errorf("处理后数据为空")
	}

	// 验证输出数据长度
	if len(decompressedData) != expectedRGBASize {
		glog.Warn("矩形", index, "RGBA数据长度不匹配，期望:", expectedRGBASize, "实际:", len(decompressedData))

		// 调整数据长度
		if len(decompressedData) > expectedRGBASize {
			glog.Warn("矩形", index, "数据过长，截断到期望长度")
			decompressedData = decompressedData[:expectedRGBASize]
		} else if len(decompressedData) < expectedRGBASize {
			glog.Warn("矩形", index, "数据过短，使用重复模式填充到期望长度")
			paddedData := make([]byte, expectedRGBASize)
			copy(paddedData, decompressedData)

			// 使用重复模式填充剩余部分，而不是用0填充
			repeatIndex := 0
			for j := len(decompressedData); j < expectedRGBASize; j++ {
				paddedData[j] = decompressedData[repeatIndex%len(decompressedData)]
				repeatIndex++
			}
			decompressedData = paddedData
		}
	}

	// 最终验证：确保数据长度正确
	if len(decompressedData) != expectedRGBASize {
		return nil, fmt.Errorf("最终数据长度仍然不正确，期望: %d, 实际: %d", expectedRGBASize, len(decompressedData))
	}

	return decompressedData, nil
}

// processWithFrontendDecompression 让前端进行解压缩处理
func (bp *BitmapProcessor) processWithFrontendDecompression(index int, rect pdu.BitmapData, targetWidth, targetHeight int) ([]byte, error) {
	// 如果让前端处理解压缩，需要将原始数据转换为RGBA格式
	// debugLogSimple("矩形", index, "让前端处理解压缩，但需要转换为RGBA格式")

	// 计算期望的未压缩数据大小（使用原始数据尺寸）
	expectedUncompressedSize := int(rect.Width) * int(rect.Height) * int(rect.BitsPerPixel) / 8

	// 计算期望的RGBA输出大小（使用目标显示尺寸）
	expectedRGBASize := targetWidth * targetHeight * 4

	// 如果数据是压缩的，需要先解压缩
	if rect.IsCompress() {
		// debugLogSimple("矩形", index, "数据是压缩的，需要先解压缩再转换")
		// 先解压缩为未压缩数据
		decompressedData := bp.decompressBitmapData(index, rect, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize)
		if len(decompressedData) == 0 {
			return nil, fmt.Errorf("解压缩失败")
		}
		return decompressedData, nil
	} else {
		// 数据未压缩，直接转换为RGBA
		// debugLogSimple("矩形", index, "数据未压缩，直接转换为RGBA格式")
		return bp.convertUncompressedData(index, rect, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize), nil
	}
}

// decompressBitmapData 解压缩位图数据
func (bp *BitmapProcessor) decompressBitmapData(index int, rect pdu.BitmapData, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize int) []byte {
	// debugLogSimple("矩形", index, "是压缩数据，在后端解压缩，原始大小:", len(rect.BitmapDataStream))

	// 验证压缩数据的基本合理性
	if len(rect.BitmapDataStream) < 4 {
		glog.Warn("矩形", index, "压缩数据太小，可能损坏")
		return nil
	}

	// 检查压缩比是否合理（压缩数据不应该比未压缩数据大）
	if len(rect.BitmapDataStream) > expectedUncompressedSize {
		glog.Warn("矩形", index, "压缩数据比未压缩数据大，可能标记错误，尝试作为未压缩数据处理")
		// 尝试作为未压缩数据处理
		return bp.convertUncompressedData(index, rect, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize)
	}

	// 在后端解压缩RLE数据
	// debugLogSimple("矩形", index, "开始RLE解压缩")

	// 创建输出缓冲区 - 使用目标尺寸
	decompressedData := make([]byte, expectedRGBASize)

	// 调用CGO解压缩函数
	var result bool
	switch rect.BitsPerPixel {
	case 15:
		result = BitmapDecompress15(
			decompressedData, targetWidth, targetHeight,
			int(rect.Width), int(rect.Height),
			rect.BitmapDataStream,
		)
	case 16:
		result = BitmapDecompress16(
			decompressedData, targetWidth, targetHeight,
			int(rect.Width), int(rect.Height),
			rect.BitmapDataStream,
		)
	case 24:
		result = BitmapDecompress24(
			decompressedData, targetWidth, targetHeight,
			int(rect.Width), int(rect.Height),
			rect.BitmapDataStream,
		)
	case 32:
		result = BitmapDecompress32(
			decompressedData, targetWidth, targetHeight,
			int(rect.Width), int(rect.Height),
			rect.BitmapDataStream,
		)
	}

	if !result {
		glog.Error("矩形", index, "RLE解压缩失败")
		// 修复：解压缩失败时，尝试更安全的处理方式
		glog.Warn("矩形", index, "尝试使用安全模式处理数据")

		// 检查原始数据是否可能已经是未压缩格式
		if len(rect.BitmapDataStream) >= expectedUncompressedSize {
			glog.Warn("矩形", index, "数据长度符合未压缩格式，尝试直接转换")
			return bp.convertUncompressedData(index, rect, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize)
		} else {
			// 数据长度不足，创建默认颜色的输出
			glog.Warn("矩形", index, "数据长度不足，创建默认颜色输出")
			defaultOutput := make([]byte, expectedRGBASize)
			for i := 0; i < len(defaultOutput); i += 4 {
				if i+3 < len(defaultOutput) {
					defaultOutput[i] = 128   // R (默认灰色)
					defaultOutput[i+1] = 128 // G
					defaultOutput[i+2] = 128 // B
					defaultOutput[i+3] = 255 // A (不透明)
				}
			}
			return defaultOutput
		}
	}

	// debugLogSimple("矩形", index, "RLE解压缩成功，解压后大小:", len(decompressedData))
	return decompressedData
}

// convertUncompressedData 转换未压缩数据
func (bp *BitmapProcessor) convertUncompressedData(index int, rect pdu.BitmapData, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize int) []byte {
	// 未压缩数据，转换为RGBA
	// debugLogSimple("矩形", index, "是未压缩数据")

	// 验证原始数据长度
	if len(rect.BitmapDataStream) != expectedUncompressedSize {
		glog.Warn("矩形", index, "未压缩数据长度不匹配")
		// glog.Warn("实际长度:", len(rect.BitmapDataStream), "期望长度:", expectedUncompressedSize)

		// 如果数据长度不匹配，尝试调整
		if len(rect.BitmapDataStream) < expectedUncompressedSize {
			glog.Warn("矩形", index, "数据不足，使用默认颜色填充")
			// 修复：使用默认颜色填充，而不是重复原始数据
			paddedData := make([]byte, expectedUncompressedSize)
			copy(paddedData, rect.BitmapDataStream)

			// 使用默认颜色填充剩余部分（灰色，避免黑色方块）
			defaultColor := byte(128) // 中等灰色
			for j := len(rect.BitmapDataStream); j < expectedUncompressedSize; j++ {
				paddedData[j] = defaultColor
			}
			rect.BitmapDataStream = paddedData
		}
		// 如果数据过多，截断到正确长度
		if len(rect.BitmapDataStream) > expectedUncompressedSize {
			// glog.Warn("矩形", index, "数据过长，截断到期望长度")
			rect.BitmapDataStream = rect.BitmapDataStream[:expectedUncompressedSize]
		}
	}

	// 修复：创建基于原始尺寸的RGBA缓冲区，用于颜色转换
	originalRGBASize := int(rect.Width) * int(rect.Height) * 4
	tempRGBA := make([]byte, originalRGBASize)

	// 添加调试信息
	// debugLogSimple("矩形", index, "开始颜色转换，位深度:", rect.BitsPerPixel, "原始尺寸:", rect.Width, "x", rect.Height)
	// debugLogSimple("矩形", index, "输入数据长度:", len(rect.BitmapDataStream), "临时缓冲区长度:", len(tempRGBA))

	// 显示输入数据的前几个字节
	if GetIsDebug() && len(rect.BitmapDataStream) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(rect.BitmapDataStream)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", rect.BitmapDataStream[j]))
		}
		glog.Debug("矩形", index, "输入数据预览:", strings.Join(previewBytes, ", "))
	}

	// 修复：使用原始数据尺寸进行颜色转换
	switch rect.BitsPerPixel {
	case 15:
		// debugLogSimple("矩形", index, "调用convert15ToRGBA")
		convert15ToRGBA(rect.BitmapDataStream, tempRGBA, int(rect.Width), int(rect.Height))
	case 16:
		// debugLogSimple("矩形", index, "调用convert16ToRGBA")
		convert16ToRGBA(rect.BitmapDataStream, tempRGBA, int(rect.Width), int(rect.Height))
	case 24:
		// debugLogSimple("矩形", index, "调用convert24ToRGBA")
		convert24ToRGBA(rect.BitmapDataStream, tempRGBA, int(rect.Width), int(rect.Height))
	case 32:
		// debugLogSimple("矩形", index, "调用convert32ToRGBA")
		convert32ToRGBA(rect.BitmapDataStream, tempRGBA, int(rect.Width), int(rect.Height))
	default:
		glog.Warn("矩形", index, "不支持的位深度:", rect.BitsPerPixel)
		return nil
	}

	// 显示临时输出数据的前几个字节
	if GetIsDebug() && len(tempRGBA) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(tempRGBA)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", tempRGBA[j]))
		}
		glog.Debug("矩形", index, "临时输出数据预览:", strings.Join(previewBytes, ", "))
	}

	// 如果原始尺寸和目标尺寸不同，需要调整数据
	if originalRGBASize != expectedRGBASize {
		glog.Debug("矩形", index, "尺寸调整: 原始RGBA大小:", originalRGBASize, "目标RGBA大小:", expectedRGBASize)

		// 创建最终输出缓冲区
		decompressedData := make([]byte, expectedRGBASize)

		// 简单的复制策略：如果目标更大，用默认颜色填充；如果目标更小，截断
		if expectedRGBASize > originalRGBASize {
			// 目标更大，复制原始数据并用默认颜色填充
			copy(decompressedData, tempRGBA)
			for i := originalRGBASize; i < expectedRGBASize; i += 4 {
				if i+3 < expectedRGBASize {
					decompressedData[i] = 128   // R (默认灰色)
					decompressedData[i+1] = 128 // G
					decompressedData[i+2] = 128 // B
					decompressedData[i+3] = 255 // A (不透明)
				}
			}
		} else {
			// 目标更小，截断原始数据
			copy(decompressedData, tempRGBA[:expectedRGBASize])
		}

		return decompressedData
	}

	// 尺寸相同，直接返回
	return tempRGBA
}

// 颜色转换函数定义
func convert15ToRGBA(src []byte, dst []byte, width, height int) {
	// debugLogSimple("convert15ToRGBA: 开始转换，输入大小:", len(src), "输出大小:", len(dst), "目标尺寸:", width, "x", height)

	// 验证输入数据
	expectedSrcSize := width * height * 2 // 15位 = 2字节/像素
	if len(src) < expectedSrcSize {
		glog.Error("convert15ToRGBA: 输入数据不足，期望:", expectedSrcSize, "实际:", len(src))
		// 如果数据不足，尝试使用可用数据
		if len(src) == 0 {
			glog.Error("convert15ToRGBA: 输入数据为空")
			return
		}
		// 计算实际可处理的像素数
		actualPixels := len(src) / 2
		actualHeight := actualPixels / width
		if actualHeight <= 0 {
			actualHeight = 1
		}
		glog.Warn("convert15ToRGBA: 调整处理尺寸为:", width, "x", actualHeight)
		height = actualHeight
	}

	// 重新计算输出缓冲区需求
	expectedDstSize := width * height * 4
	if len(dst) < expectedDstSize {
		glog.Error("convert15ToRGBA: 输出缓冲区不足，期望:", expectedDstSize, "实际:", len(dst))
		return
	}

	// 显示输入数据的前几个字节用于调试
	if GetIsDebug() && len(src) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(src)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", src[j]))
		}
		glog.Debug("convert15ToRGBA: 输入数据预览:", strings.Join(previewBytes, ", "))
	}

	// 确保不超出边界
	maxPixels := min(len(src)/2, len(dst)/4)
	actualHeight := min(height, maxPixels/width)
	if actualHeight <= 0 {
		actualHeight = 1
	}

	// debugLogSimple("convert15ToRGBA: 开始处理像素，实际高度:", actualHeight, "宽度:", width)

	for y := 0; y < actualHeight; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if i*2+1 >= len(src) || i*4+3 >= len(dst) {
				// 修复：超出边界时使用默认颜色，而不是跳过
				if i*4+3 < len(dst) {
					dst[i*4+0] = 128 // R (默认灰色)
					dst[i*4+1] = 128 // G
					dst[i*4+2] = 128 // B
					dst[i*4+3] = 255 // A (不透明)
				}
				continue
			}

			// 修复：RDP使用小端序，低字节在前
			val := uint16(src[i*2]) | uint16(src[i*2+1])<<8

			// 修复：使用与core/io.go中RGB555ToRGB函数完全相同的转换方式
			// RGB555格式：RRRRRGGGGGBBBBB (5-5-5)
			r := uint8((val & 0x7C00) >> 7) // 5 bits red -> 8 bits
			g := uint8((val & 0x03E0) >> 2) // 5 bits green -> 8 bits
			b := uint8((val & 0x001F) << 3) // 5 bits blue -> 8 bits

			// 输出RGBA格式
			dst[i*4+0] = r   // R
			dst[i*4+1] = g   // G
			dst[i*4+2] = b   // B
			dst[i*4+3] = 255 // A

			// 调试第一个像素
			if x == 0 && y == 0 {
				glog.Debug("convert15ToRGBA: 第一个像素调试信息:")
				glog.Debug("  原始字节:", fmt.Sprintf("0x%02x 0x%02x", src[i*2], src[i*2+1]))
				glog.Debug("  小端序值:", fmt.Sprintf("0x%04X", val))
				glog.Debug("  RGB555: R=%d G=%d B=%d", (val&0x7C00)>>10, (val&0x03E0)>>5, val&0x001F)
				glog.Debug("  8位转换: R=%d G=%d B=%d", r, g, b)
			}
		}
	}

	// 显示输出数据的前几个字节用于调试
	if GetIsDebug() && len(dst) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(dst)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", dst[j]))
		}
		glog.Debug("convert15ToRGBA: 输出数据预览:", strings.Join(previewBytes, ", "))
	}

	// 新增：检测全黑输出
	detectBlackOutput("convert15ToRGBA", dst, width, height)

	// debugLogSimple("convert15ToRGBA: 转换完成，实际处理:", actualHeight, "行")
}

func convert16ToRGBA(src []byte, dst []byte, width, height int) {
	// debugLogSimple("convert16ToRGBA: 开始转换，输入大小:", len(src), "输出大小:", len(dst), "目标尺寸:", width, "x", height)

	// 验证输入数据
	expectedSrcSize := width * height * 2 // 16位 = 2字节/像素
	if len(src) < expectedSrcSize {
		glog.Error("convert16ToRGBA: 输入数据不足，期望:", expectedSrcSize, "实际:", len(src))
		// 如果数据不足，尝试使用可用数据
		if len(src) == 0 {
			glog.Error("convert16ToRGBA: 输入数据为空")
			return
		}
		// 计算实际可处理的像素数
		actualPixels := len(src) / 2
		actualHeight := actualPixels / width
		if actualHeight <= 0 {
			actualHeight = 1
		}
		glog.Warn("convert16ToRGBA: 调整处理尺寸为:", width, "x", actualHeight)
		height = actualHeight
	}

	// 重新计算输出缓冲区需求
	expectedDstSize := width * height * 4
	if len(dst) < expectedDstSize {
		glog.Error("convert16ToRGBA: 输出缓冲区不足，期望:", expectedDstSize, "实际:", len(dst))
		return
	}

	// 显示输入数据的前几个字节用于调试
	if GetIsDebug() && len(src) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(src)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", src[j]))
		}
		glog.Debug("convert16ToRGBA: 输入数据预览:", strings.Join(previewBytes, ", "))
	}

	// 确保不超出边界
	maxPixels := min(len(src)/2, len(dst)/4)
	actualHeight := min(height, maxPixels/width)
	if actualHeight <= 0 {
		actualHeight = 1
	}

	// debugLogSimple("convert16ToRGBA: 开始处理像素，实际高度:", actualHeight, "宽度:", width)

	for y := 0; y < actualHeight; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if i*2+1 >= len(src) || i*4+3 >= len(dst) {
				continue
			}

			// 修复：RDP使用小端序，低字节在前
			val := uint16(src[i*2]) | uint16(src[i*2+1])<<8

			// 修复：使用与core/io.go中RGB565ToRGB函数完全相同的转换方式
			// RGB565格式：RRRRRGGGGGGBBBBB (5-6-5)
			r := uint8((val & 0xF800) >> 8) // 5 bits red -> 8 bits
			g := uint8((val & 0x07E0) >> 3) // 6 bits green -> 8 bits
			b := uint8((val & 0x001F) << 3) // 5 bits blue -> 8 bits

			// 输出RGBA格式
			dst[i*4+0] = r   // R
			dst[i*4+1] = g   // G
			dst[i*4+2] = b   // B
			dst[i*4+3] = 255 // A

			// 调试第一个像素
			if x == 0 && y == 0 {
				glog.Debug("convert16ToRGBA: 第一个像素调试信息:")
				glog.Debug("  原始字节:", fmt.Sprintf("0x%02x 0x%02x", src[i*2], src[i*2+1]))
				glog.Debug("  小端序值:", fmt.Sprintf("0x%04X", val))
				glog.Debug("  RGB565: R=%d G=%d B=%d", (val&0xF800)>>11, (val&0x07E0)>>5, val&0x001F)
				glog.Debug("  8位转换: R=%d G=%d B=%d", r, g, b)
			}
		}
	}

	// 显示输出数据的前几个字节用于调试
	if GetIsDebug() && len(dst) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(dst)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", dst[j]))
		}
		glog.Debug("convert16ToRGBA: 输出数据预览:", strings.Join(previewBytes, ", "))
	}

	// 新增：检测全黑输出
	detectBlackOutput("convert16ToRGBA", dst, width, height)

	// debugLogSimple("convert16ToRGBA: 转换完成，实际处理:", actualHeight, "行")
}

func convert24ToRGBA(src []byte, dst []byte, width, height int) {
	// debugLogSimple("convert24ToRGBA: 开始转换，输入大小:", len(src), "输出大小:", len(dst), "目标尺寸:", width, "x", height)

	// 验证输入数据
	expectedSrcSize := width * height * 3 // 24位 = 3字节/像素
	if len(src) < expectedSrcSize {
		glog.Error("convert24ToRGBA: 输入数据不足，期望:", expectedSrcSize, "实际:", len(src))
		// 如果数据不足，尝试使用可用数据
		if len(src) == 0 {
			glog.Error("convert24ToRGBA: 输入数据为空")
			return
		}
		// 计算实际可处理的像素数
		actualPixels := len(src) / 3
		actualHeight := actualPixels / width
		if actualHeight <= 0 {
			actualHeight = 1
		}
		glog.Warn("convert24ToRGBA: 调整处理尺寸为:", width, "x", actualHeight)
		height = actualHeight
	}

	// 重新计算输出缓冲区需求
	expectedDstSize := width * height * 4
	if len(dst) < expectedDstSize {
		glog.Error("convert24ToRGBA: 输出缓冲区不足，期望:", expectedDstSize, "实际:", len(dst))
		return
	}

	// 显示输入数据的前几个字节用于调试
	if GetIsDebug() && len(src) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(src)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", src[j]))
		}
		glog.Debug("convert24ToRGBA: 输入数据预览:", strings.Join(previewBytes, ", "))
	}

	// 确保不超出边界
	maxPixels := min(len(src)/3, len(dst)/4)
	actualHeight := min(height, maxPixels/width)
	if actualHeight <= 0 {
		actualHeight = 1
	}

	// 移除：不初始化输出缓冲区为0，让颜色转换函数直接写入数据
	// 初始化输出缓冲区为0
	// for i := 0; i < len(dst); i++ {
	// 	dst[i] = 0
	// }

	// debugLogSimple("convert24ToRGBA: 开始处理像素，实际高度:", actualHeight, "宽度:", width)

	for y := 0; y < actualHeight; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if i*3+2 >= len(src) || i*4+3 >= len(dst) {
				continue
			}
			// 修复：RDP使用BGR顺序，需要转换为RGB
			// BGR顺序：B=src[i*3+0], G=src[i*3+1], R=src[i*3+2]
			// 转换为RGBA：R=dst[i*4+0], G=dst[i*4+1], B=dst[i*4+2], A=dst[i*4+3]
			dst[i*4+0] = src[i*3+2] // R (原BGR中的R)
			dst[i*4+1] = src[i*3+1] // G (原BGR中的G)
			dst[i*4+2] = src[i*3+0] // B (原BGR中的B)
			dst[i*4+3] = 255        // A (完全不透明)
		}
	}

	// 显示输出数据的前几个字节用于调试
	if GetIsDebug() && len(dst) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(dst)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", dst[j]))
		}
		glog.Debug("convert24ToRGBA: 输出数据预览:", strings.Join(previewBytes, ", "))
	}

	// 新增：检测全黑输出
	detectBlackOutput("convert24ToRGBA", dst, width, height)

	// debugLogSimple("convert24ToRGBA: 转换完成，实际处理:", actualHeight, "行")
}

func convert32ToRGBA(src []byte, dst []byte, width, height int) {
	// debugLogSimple("convert32ToRGBA: 开始转换，输入大小:", len(src), "输出大小:", len(dst), "目标尺寸:", width, "x", height)

	// 验证输入数据
	expectedSrcSize := width * height * 4 // 32位 = 4字节/像素
	if len(src) < expectedSrcSize {
		glog.Error("convert32ToRGBA: 输入数据不足，期望:", expectedSrcSize, "实际:", len(src))
		// 如果数据不足，尝试使用可用数据
		if len(src) == 0 {
			glog.Error("convert32ToRGBA: 输入数据为空")
			return
		}
		// 计算实际可处理的像素数
		actualPixels := len(src) / 4
		actualHeight := actualPixels / width
		if actualHeight <= 0 {
			actualHeight = 1
		}
		glog.Warn("convert32ToRGBA: 调整处理尺寸为:", width, "x", actualHeight)
		height = actualHeight
	}

	// 重新计算输出缓冲区需求
	expectedDstSize := width * height * 4
	if len(dst) < expectedDstSize {
		glog.Error("convert32ToRGBA: 输出缓冲区不足，期望:", expectedDstSize, "实际:", len(dst))
		return
	}

	// 显示输入数据的前几个字节用于调试
	if GetIsDebug() && len(src) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(src)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", src[j]))
		}
		glog.Debug("convert32ToRGBA: 输入数据预览:", strings.Join(previewBytes, ", "))
	}

	// 确保不超出边界
	maxPixels := min(len(src)/4, len(dst)/4)
	actualHeight := min(height, maxPixels/width)
	if actualHeight <= 0 {
		actualHeight = 1
	}

	// 移除：不初始化输出缓冲区为0，让颜色转换函数直接写入数据
	// 初始化输出缓冲区为0
	// for i := 0; i < len(dst); i++ {
	// 	dst[i] = 0
	// }

	// debugLogSimple("convert32ToRGBA: 开始处理像素，实际高度:", actualHeight, "宽度:", width)

	for y := 0; y < actualHeight; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if i*4+3 >= len(src) || i*4+3 >= len(dst) {
				continue
			}
			// 修复：RDP使用BGRA顺序，需要转换为RGBA
			// BGRA顺序：B=src[i*4+0], G=src[i*4+1], R=src[i*4+2], A=src[i*4+3]
			// 转换为RGBA：R=dst[i*4+0], G=dst[i*4+1], B=dst[i*4+2], A=dst[i*4+3]
			dst[i*4+0] = src[i*4+2] // R (原BGRA中的R)
			dst[i*4+1] = src[i*4+1] // G (原BGRA中的G)
			dst[i*4+2] = src[i*4+0] // B (原BGRA中的B)
			dst[i*4+3] = src[i*4+3] // A (原BGRA中的A)
		}
	}

	// 显示输出数据的前几个字节用于调试
	if GetIsDebug() && len(dst) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(dst)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", dst[j]))
		}
		glog.Debug("convert32ToRGBA: 输出数据预览:", strings.Join(previewBytes, ", "))
	}

	// 新增：检测全黑输出
	detectBlackOutput("convert32ToRGBA", dst, width, height)

	// debugLogSimple("convert32ToRGBA: 转换完成，实际处理:", actualHeight, "行")
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// detectBlackOutput 检测全黑输出数据（独立函数）
func detectBlackOutput(functionName string, output []byte, width, height int) {
	if len(output) == 0 {
		return
	}

	// 检查RGBA数据是否全为0（全黑）
	allBlack := true
	sampleCount := 0
	nonZeroCount := 0
	maxSample := min(400, len(output)) // 最多检查100个像素（400字节）

	for i := 0; i < maxSample; i += 4 { // 每4字节一个像素
		if i+3 < len(output) {
			// 检查RGB通道是否都为0（忽略Alpha通道）
			if output[i] != 0 || output[i+1] != 0 || output[i+2] != 0 {
				nonZeroCount++
				// 如果发现非零像素，检查是否足够多
				if nonZeroCount > 10 { // 允许最多10个非零像素
					allBlack = false
					break
				}
			}
			sampleCount++
		}
	}

	if allBlack && sampleCount > 0 {
		glog.Warn("⚠️ 检测到全黑输出！函数:", functionName, "详情:", map[string]interface{}{
			"width":        width,
			"height":       height,
			"outputSize":   len(output),
			"sampleCount":  sampleCount,
			"nonZeroCount": nonZeroCount,
		})

		// 显示前20个字节的详细信息
		previewBytes := make([]string, 0)
		for j := 0; j < min(20, len(output)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", output[j]))
		}
		glog.Warn("函数", functionName, "全黑输出预览:", strings.Join(previewBytes, ", "))
	}
}
