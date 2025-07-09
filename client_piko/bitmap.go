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
	decompressOnBackend bool // 控制是否在后端解压缩
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

// HandleBitmapUpdate 处理位图更新
func (bp *BitmapProcessor) HandleBitmapUpdate(rectangles []pdu.BitmapData) {
	debugLogSimple("HandleBitmapUpdate被调用，矩形数量:", len(rectangles))

	if bp.webServer == nil {
		glog.Warn("WebServer未设置，无法处理位图更新")
		return
	}

	debugLogSimple("处理位图更新，矩形数量:", len(rectangles))

	// 处理位图数据
	var bitmapData []map[string]interface{}

	for i, rect := range rectangles {
		processedData := bp.processRectangle(i, rect)
		if processedData != nil {
			bitmapData = append(bitmapData, processedData)
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

	debugLogSimple("准备广播位图更新事件，矩形数量:", len(bitmapData))
	bp.webServer.BroadcastRDPUpdate(updateData)
	debugLogSimple("位图更新事件广播完成")
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

	return true
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
			glog.Warn("矩形", index, "数据过短，填充到期望长度")
			paddedData := make([]byte, expectedRGBASize)
			copy(paddedData, decompressedData)
			// 用0填充剩余部分
			for j := len(decompressedData); j < expectedRGBASize; j++ {
				paddedData[j] = 0
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
	debugLogSimple("矩形", index, "让前端处理解压缩，但需要转换为RGBA格式")

	// 计算期望的未压缩数据大小（使用原始数据尺寸）
	expectedUncompressedSize := int(rect.Width) * int(rect.Height) * int(rect.BitsPerPixel) / 8

	// 计算期望的RGBA输出大小（使用目标显示尺寸）
	expectedRGBASize := targetWidth * targetHeight * 4

	// 如果数据是压缩的，需要先解压缩
	if rect.IsCompress() {
		debugLogSimple("矩形", index, "数据是压缩的，需要先解压缩再转换")
		// 先解压缩为未压缩数据
		decompressedData := bp.decompressBitmapData(index, rect, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize)
		if len(decompressedData) == 0 {
			return nil, fmt.Errorf("解压缩失败")
		}
		return decompressedData, nil
	} else {
		// 数据未压缩，直接转换为RGBA
		debugLogSimple("矩形", index, "数据未压缩，直接转换为RGBA格式")
		return bp.convertUncompressedData(index, rect, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize), nil
	}
}

// decompressBitmapData 解压缩位图数据
func (bp *BitmapProcessor) decompressBitmapData(index int, rect pdu.BitmapData, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize int) []byte {
	debugLogSimple("矩形", index, "是压缩数据，在后端解压缩，原始大小:", len(rect.BitmapDataStream))

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
	debugLogSimple("矩形", index, "开始RLE解压缩")

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
		// 解压缩失败，尝试作为未压缩数据处理
		debugLogSimple("矩形", index, "解压缩失败，尝试作为未压缩数据处理")
		return bp.convertUncompressedData(index, rect, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize)
	}

	debugLogSimple("矩形", index, "RLE解压缩成功，解压后大小:", len(decompressedData))
	return decompressedData
}

// convertUncompressedData 转换未压缩数据
func (bp *BitmapProcessor) convertUncompressedData(index int, rect pdu.BitmapData, targetWidth, targetHeight, expectedUncompressedSize, expectedRGBASize int) []byte {
	// 未压缩数据，转换为RGBA
	debugLogSimple("矩形", index, "是未压缩数据")

	// 验证原始数据长度
	if len(rect.BitmapDataStream) != expectedUncompressedSize {
		glog.Warn("矩形", index, "未压缩数据长度不匹配")
		glog.Warn("实际长度:", len(rect.BitmapDataStream), "期望长度:", expectedUncompressedSize)

		// 如果数据长度不匹配，尝试调整或跳过
		if len(rect.BitmapDataStream) < expectedUncompressedSize {
			glog.Warn("矩形", index, "数据不足，跳过")
			return nil
		}
		// 如果数据过多，截断到正确长度
		if len(rect.BitmapDataStream) > expectedUncompressedSize {
			glog.Warn("矩形", index, "数据过多，截断到正确长度")
			rect.BitmapDataStream = rect.BitmapDataStream[:expectedUncompressedSize]
		}
	}

	decompressedData := make([]byte, expectedRGBASize)

	// 添加调试信息
	debugLogSimple("矩形", index, "开始颜色转换，位深度:", rect.BitsPerPixel, "目标尺寸:", targetWidth, "x", targetHeight)
	debugLogSimple("矩形", index, "输入数据长度:", len(rect.BitmapDataStream), "输出缓冲区长度:", len(decompressedData))

	// 显示输入数据的前几个字节
	if GetIsDebug() && len(rect.BitmapDataStream) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(rect.BitmapDataStream)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", rect.BitmapDataStream[j]))
		}
		glog.Debug("矩形", index, "输入数据预览:", strings.Join(previewBytes, ", "))
	}

	// 修复：使用原始数据尺寸进行颜色转换，但输出到目标尺寸的缓冲区
	switch rect.BitsPerPixel {
	case 15:
		debugLogSimple("矩形", index, "调用convert15ToRGBA")
		convert15ToRGBA(rect.BitmapDataStream, decompressedData, int(rect.Width), int(rect.Height))
	case 16:
		debugLogSimple("矩形", index, "调用convert16ToRGBA")
		convert16ToRGBA(rect.BitmapDataStream, decompressedData, int(rect.Width), int(rect.Height))
	case 24:
		debugLogSimple("矩形", index, "调用convert24ToRGBA")
		convert24ToRGBA(rect.BitmapDataStream, decompressedData, int(rect.Width), int(rect.Height))
	case 32:
		debugLogSimple("矩形", index, "调用convert32ToRGBA")
		convert32ToRGBA(rect.BitmapDataStream, decompressedData, int(rect.Width), int(rect.Height))
	default:
		glog.Warn("矩形", index, "不支持的位深度:", rect.BitsPerPixel)
		return nil
	}

	// 显示输出数据的前几个字节
	if GetIsDebug() && len(decompressedData) > 0 {
		previewBytes := make([]string, 0)
		for j := 0; j < min(16, len(decompressedData)); j++ {
			previewBytes = append(previewBytes, fmt.Sprintf("0x%02x", decompressedData[j]))
		}
		glog.Debug("矩形", index, "输出数据预览:", strings.Join(previewBytes, ", "))
	}

	debugLogSimple("矩形", index, "未压缩数据转换成功，原始大小:", len(rect.BitmapDataStream), "转换后大小:", len(decompressedData))
	return decompressedData
}

// 颜色转换函数定义
func convert15ToRGBA(src []byte, dst []byte, width, height int) {
	debugLogSimple("convert15ToRGBA: 开始转换，输入大小:", len(src), "输出大小:", len(dst), "目标尺寸:", width, "x", height)

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

	// 初始化输出缓冲区为0
	for i := 0; i < len(dst); i++ {
		dst[i] = 0
	}

	debugLogSimple("convert15ToRGBA: 开始处理像素，实际高度:", actualHeight, "宽度:", width)

	for y := 0; y < actualHeight; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if i*2+1 >= len(src) || i*4+3 >= len(dst) {
				continue
			}
			// 修复：RDP使用小端序，所以低字节在前
			val := uint16(src[i*2]) | uint16(src[i*2+1])<<8
			r := uint8((val&0x7c00)>>10) * 255 / 31
			g := uint8((val&0x03e0)>>5) * 255 / 31
			b := uint8(val&0x001f) * 255 / 31
			// 修复：确保正确的RGBA顺序
			dst[i*4+0] = r
			dst[i*4+1] = g
			dst[i*4+2] = b
			dst[i*4+3] = 255
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

	debugLogSimple("convert15ToRGBA: 转换完成，实际处理:", actualHeight, "行")
}

func convert16ToRGBA(src []byte, dst []byte, width, height int) {
	debugLogSimple("convert16ToRGBA: 开始转换，输入大小:", len(src), "输出大小:", len(dst), "目标尺寸:", width, "x", height)

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

	// 初始化输出缓冲区为0
	for i := 0; i < len(dst); i++ {
		dst[i] = 0
	}

	debugLogSimple("convert16ToRGBA: 开始处理像素，实际高度:", actualHeight, "宽度:", width)

	for y := 0; y < actualHeight; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if i*2+1 >= len(src) || i*4+3 >= len(dst) {
				continue
			}
			// 修复：RDP使用小端序，所以低字节在前
			val := uint16(src[i*2]) | uint16(src[i*2+1])<<8
			r := uint8((val&0xf800)>>11) * 255 / 31
			g := uint8((val&0x07e0)>>5) * 255 / 63
			b := uint8(val&0x001f) * 255 / 31
			// 修复：确保正确的RGBA顺序
			dst[i*4+0] = r
			dst[i*4+1] = g
			dst[i*4+2] = b
			dst[i*4+3] = 255
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

	debugLogSimple("convert16ToRGBA: 转换完成，实际处理:", actualHeight, "行")
}

func convert24ToRGBA(src []byte, dst []byte, width, height int) {
	debugLogSimple("convert24ToRGBA: 开始转换，输入大小:", len(src), "输出大小:", len(dst), "目标尺寸:", width, "x", height)

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

	// 初始化输出缓冲区为0
	for i := 0; i < len(dst); i++ {
		dst[i] = 0
	}

	debugLogSimple("convert24ToRGBA: 开始处理像素，实际高度:", actualHeight, "宽度:", width)

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

	debugLogSimple("convert24ToRGBA: 转换完成，实际处理:", actualHeight, "行")
}

func convert32ToRGBA(src []byte, dst []byte, width, height int) {
	debugLogSimple("convert32ToRGBA: 开始转换，输入大小:", len(src), "输出大小:", len(dst), "目标尺寸:", width, "x", height)

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

	// 初始化输出缓冲区为0
	for i := 0; i < len(dst); i++ {
		dst[i] = 0
	}

	debugLogSimple("convert32ToRGBA: 开始处理像素，实际高度:", actualHeight, "宽度:", width)

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

	debugLogSimple("convert32ToRGBA: 转换完成，实际处理:", actualHeight, "行")
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
