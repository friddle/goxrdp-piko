package client_piko

/*
#cgo CFLAGS: -I${SRCDIR}/c
#include "rle.h"
#include "rle.c"
*/
import "C"
import (
	"unsafe"
)

// bitmapDecompress15 解压缩15位色深的位图，返回int值
func bitmapDecompress15(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) int {
	if len(output) == 0 || len(input) == 0 {
		return 0
	}

	// 验证输出缓冲区大小是否正确 (15位 = 2字节/像素，转换为32位RGBA = 4字节/像素)
	expectedOutputSize := outputWidth * outputHeight * 4
	if len(output) < expectedOutputSize {
		return 0
	}

	result := C.bitmap_decompress_15(
		(*C.uchar)(unsafe.Pointer(&output[0])),
		C.int(outputWidth),
		C.int(outputHeight),
		C.int(inputWidth),
		C.int(inputHeight),
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.int(len(input)),
	)

	return int(result)
}

// bitmapDecompress16 解压缩16位色深的位图，返回int值
func bitmapDecompress16(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) int {
	if len(output) == 0 || len(input) == 0 {
		return 0
	}

	// 验证输出缓冲区大小是否正确 (16位 = 2字节/像素，转换为32位RGBA = 4字节/像素)
	expectedOutputSize := outputWidth * outputHeight * 4
	if len(output) < expectedOutputSize {
		return 0
	}

	result := C.bitmap_decompress_16(
		(*C.uchar)(unsafe.Pointer(&output[0])),
		C.int(outputWidth),
		C.int(outputHeight),
		C.int(inputWidth),
		C.int(inputHeight),
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.int(len(input)),
	)

	return int(result)
}

// bitmapDecompress24 解压缩24位色深的位图，返回int值
func bitmapDecompress24(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) int {
	if len(output) == 0 || len(input) == 0 {
		return 0
	}

	// 验证输出缓冲区大小是否正确 (24位 = 3字节/像素，转换为32位RGBA = 4字节/像素)
	expectedOutputSize := outputWidth * outputHeight * 4
	if len(output) < expectedOutputSize {
		return 0
	}

	result := C.bitmap_decompress_24(
		(*C.uchar)(unsafe.Pointer(&output[0])),
		C.int(outputWidth),
		C.int(outputHeight),
		C.int(inputWidth),
		C.int(inputHeight),
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.int(len(input)),
	)

	return int(result)
}

// bitmapDecompress32 解压缩32位色深的位图，返回int值
func bitmapDecompress32(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) int {
	if len(output) == 0 || len(input) == 0 {
		return 0
	}

	// 验证输出缓冲区大小是否正确 (32位 = 4字节/像素)
	expectedOutputSize := outputWidth * outputHeight * 4
	if len(output) < expectedOutputSize {
		return 0
	}

	result := C.bitmap_decompress_32(
		(*C.uchar)(unsafe.Pointer(&output[0])),
		C.int(outputWidth),
		C.int(outputHeight),
		C.int(inputWidth),
		C.int(inputHeight),
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.int(len(input)),
	)

	return int(result)
}

// BitmapDecompress15 解压缩15位色深的位图
func BitmapDecompress15(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) bool {
	result := bitmapDecompress15(output, outputWidth, outputHeight, inputWidth, inputHeight, input)
	return result != 0
}

// BitmapDecompress16 解压缩16位色深的位图
func BitmapDecompress16(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) bool {
	result := bitmapDecompress16(output, outputWidth, outputHeight, inputWidth, inputHeight, input)
	return result != 0
}

// BitmapDecompress24 解压缩24位色深的位图
func BitmapDecompress24(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) bool {
	result := bitmapDecompress24(output, outputWidth, outputHeight, inputWidth, inputHeight, input)
	return result != 0
}

// BitmapDecompress32 解压缩32位色深的位图
func BitmapDecompress32(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) bool {
	result := bitmapDecompress32(output, outputWidth, outputHeight, inputWidth, inputHeight, input)
	return result != 0
}
