# RLE解压缩库

基于Go代码逻辑的纯JavaScript RLE（Run-Length Encoding）解压缩库，专为RDP（Remote Desktop Protocol）位图解压缩设计。

## 功能特性

- ✅ **多格式支持**：支持15位(RGB555)、16位(RGB565)、24位(BGR)、32位(BGRA)位图解压缩
- ✅ **字节序正确**：使用小端序（低字节在前），符合RDP协议规范
- ✅ **颜色转换**：自动将RDP的BGR/BGRA格式转换为Web标准的RGBA格式
- ✅ **性能优化**：使用TypedArray提高性能，支持大图像处理
- ✅ **错误处理**：完善的错误检查和异常处理机制
- ✅ **跨平台**：支持Node.js和浏览器环境
- ✅ **零依赖**：纯JavaScript实现，无需外部依赖

## 快速开始

### 浏览器环境

```html
<!DOCTYPE html>
<html>
<head>
    <title>RLE解压缩示例</title>
</head>
<body>
    <canvas id="canvas" width="800" height="600"></canvas>
    
    <script src="rle-decompress.js"></script>
    <script>
        // 创建解压缩器实例
        const decompressor = new RLEDecompressor();
        
        // 解压缩15位RGB555数据
        const output = new Uint8Array(width * height * 4); // RGBA格式
        const success = decompressor.bitmapDecompress15(
            output,           // 输出缓冲区
            outputWidth,      // 输出宽度
            outputHeight,     // 输出高度
            inputWidth,       // 输入宽度
            inputHeight,      // 输入高度
            compressedData    // 压缩的输入数据
        );
        
        if (success) {
            // 将结果绘制到Canvas
            const canvas = document.getElementById('canvas');
            const ctx = canvas.getContext('2d');
            const imageData = ctx.createImageData(width, height);
            imageData.data.set(output);
            ctx.putImageData(imageData, 0, 0);
        }
    </script>
</body>
</html>
```

### Node.js环境

```javascript
const RLEDecompressor = require('./rle-decompress.js');

const decompressor = new RLEDecompressor();

// 解压缩数据
const output = new Uint8Array(width * height * 4);
const success = decompressor.bitmapDecompress15(
    output, width, height, inputWidth, inputHeight, compressedData
);

if (success) {
    console.log('解压缩成功');
    // 处理output数据...
}
```

## API文档

### 构造函数

```javascript
const decompressor = new RLEDecompressor();
```

### 主要方法

#### bitmapDecompress15(output, outputWidth, outputHeight, inputWidth, inputHeight, input)

解压缩15位(RGB555)位图数据到RGBA格式。

**参数：**
- `output` (Uint8Array): 输出RGBA缓冲区
- `outputWidth` (number): 输出宽度
- `outputHeight` (number): 输出高度
- `inputWidth` (number): 输入宽度
- `inputHeight` (number): 输入高度
- `input` (Uint8Array): 输入压缩数据

**返回值：** `boolean` - 是否成功

#### bitmapDecompress16(output, outputWidth, outputHeight, inputWidth, inputHeight, input)

解压缩16位(RGB565)位图数据到RGBA格式。

**参数：** 同上

**返回值：** `boolean` - 是否成功

#### bitmapDecompress24(output, outputWidth, outputHeight, inputWidth, inputHeight, input)

解压缩24位(BGR)位图数据到RGBA格式。

**参数：** 同上

**返回值：** `boolean` - 是否成功

#### bitmapDecompress32(output, outputWidth, outputHeight, inputWidth, inputHeight, input)

解压缩32位(BGRA)位图数据到RGBA格式。

**参数：** 同上

**返回值：** `boolean` - 是否成功

#### debugColorConversion()

输出调试信息到控制台。

## 格式说明

### RGB555 (15位)
- 格式：`RRRRRGGGGGGBBBBB`
- 位分配：5位红、5位绿、5位蓝
- 字节序：小端序（低字节在前）

### RGB565 (16位)
- 格式：`RRRRRGGGGGGBBBBB`
- 位分配：5位红、6位绿、5位蓝
- 字节序：小端序（低字节在前）

### BGR24 (24位)
- 格式：`BGR`（蓝绿红）
- 位分配：8位蓝、8位绿、8位红
- 字节序：小端序（低字节在前）

### BGRA32 (32位)
- 格式：`BGRA`（蓝绿红透明）
- 位分配：8位蓝、8位绿、8位红、8位透明
- 字节序：小端序（低字节在前）

## 技术细节

### 字节序处理
RDP协议使用小端序（Little-Endian），即低字节在前。本库正确处理了字节序转换：

```javascript
// 小端序读取16位值
const pixel = temp[pixelIdx] | (temp[pixelIdx + 1] << 8);
```

### 颜色格式转换
RDP使用BGR/BGRA颜色顺序，而Web标准使用RGB/RGBA。本库自动进行转换：

```javascript
// BGR -> RGB转换
const b = temp[srcIdx + 0]; // B
const g = temp[srcIdx + 1]; // G  
const r = temp[srcIdx + 2]; // R

output[dstIdx + 0] = r; // R
output[dstIdx + 1] = g; // G
output[dstIdx + 2] = b; // B
```

### 性能优化
- 使用`Uint8Array`和`Uint16Array`提高内存访问效率
- 实现REPEAT宏优化循环性能
- 支持大图像处理，内存使用优化

## 错误处理

库包含完善的错误处理机制：

```javascript
try {
    const success = decompressor.bitmapDecompress15(output, width, height, inputWidth, inputHeight, input);
    if (success) {
        console.log('解压缩成功');
    } else {
        console.error('解压缩失败');
    }
} catch (error) {
    console.error('解压缩错误:', error.message);
}
```

## 测试

运行示例页面进行测试：

```bash
# 在浏览器中打开
open rle-example.html
```

示例页面包含：
- 各种格式的解压缩测试
- 实时Canvas显示
- 错误状态反馈
- 使用示例代码

## 兼容性

- **浏览器**：支持所有现代浏览器（ES6+）
- **Node.js**：支持Node.js 8.0+
- **移动端**：支持移动浏览器

## 许可证

本项目基于MIT许可证开源。

## 贡献

欢迎提交Issue和Pull Request来改进这个库。

## 更新日志

### v1.0.0
- 初始版本发布
- 支持15位、16位、24位、32位位图解压缩
- 完整的RDP协议兼容性
- 浏览器和Node.js环境支持 