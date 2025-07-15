package client_piko

import (
	"fmt"
)

// Bitmap decompression routines
// Copyright (C) Matthew Chapman <matthewc.unsw.edu.au> 1999-2008
// Ported from C to Go

const (
	noMaskUpdate = iota
	insertFillOrMix
	insertFill
	insertMix
)

// Constants for handling bitmap decompression
const (
	fomMask1 = 0x03
	fomMask2 = 0x05
)

// BitmapDecompress15 decompresses 15-bit (RGB555) bitmap data to RGBA format
func BitmapDecompress15(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) bool {
	// Allocate temporary buffer for decompressed 15-bit data
	temp := make([]byte, inputWidth*inputHeight*2)

	// Decompress into temp buffer
	if !bitmapDecompress2(temp, inputWidth, inputHeight, input) {
		return false
	}

	// Convert to RGBA - 修复：RDP使用小端序
	for y := 0; y < outputHeight && y < inputHeight; y++ {
		for x := 0; x < outputWidth && x < inputWidth; x++ {
			pixelIdx := (y*inputWidth + x) * 2

			// 修复：RDP使用小端序，低字节在前
			pixel := uint16(temp[pixelIdx]) | uint16(temp[pixelIdx+1])<<8

			// 修复：使用与core/io.go中RGB555ToRGB函数相同的转换方式
			// RGB555: RRRRRGGGGGGBBBBB (bits 15-11, 10-6, 5-1, bit 0 unused)
			r := uint8(pixel & 0x7C00 >> 7) // 5 bits red -> 8 bits
			g := uint8(pixel & 0x03E0 >> 2) // 5 bits green -> 8 bits
			b := uint8(pixel & 0x001F << 3) // 5 bits blue -> 8 bits

			// 输出RGBA格式
			idx := (y*outputWidth + x) * 4
			output[idx+0] = r   // R
			output[idx+1] = g   // G
			output[idx+2] = b   // B
			output[idx+3] = 255 // A
		}
	}

	return true
}

// BitmapDecompress16 decompresses 16-bit (RGB565) bitmap data to RGBA format
func BitmapDecompress16(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) bool {
	// Allocate temporary buffer for decompressed 16-bit data
	temp := make([]byte, inputWidth*inputHeight*2)

	// Decompress into temp buffer
	if !bitmapDecompress2(temp, inputWidth, inputHeight, input) {
		return false
	}

	// Convert to RGBA - 修复：RDP使用小端序
	for y := 0; y < outputHeight && y < inputHeight; y++ {
		for x := 0; x < outputWidth && x < inputWidth; x++ {
			pixelIdx := (y*inputWidth + x) * 2

			// 修复：RDP使用小端序，低字节在前
			pixel := uint16(temp[pixelIdx]) | uint16(temp[pixelIdx+1])<<8

			// 修复：使用与core/io.go中RGB565ToRGB函数相同的转换方式
			// RGB565: RRRRRGGGGGGBBBBB (bits 15-11, 10-5, 4-0)
			r := uint8(pixel & 0xF800 >> 8) // 5 bits red -> 8 bits
			g := uint8(pixel & 0x07E0 >> 3) // 6 bits green -> 8 bits
			b := uint8(pixel & 0x001F << 3) // 5 bits blue -> 8 bits

			// 输出RGBA格式
			idx := (y*outputWidth + x) * 4
			output[idx+0] = r   // R
			output[idx+1] = g   // G
			output[idx+2] = b   // B
			output[idx+3] = 255 // A
		}
	}
	return true
}

// BitmapDecompress24 decompresses 24-bit BGR bitmap data to RGBA format
func BitmapDecompress24(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) bool {
	// Allocate temporary buffer for decompressed 24-bit data
	temp := make([]byte, inputWidth*inputHeight*3)

	// Decompress into temp buffer
	if !bitmapDecompress3(temp, inputWidth, inputHeight, input) {
		return false
	}

	// Convert BGR to RGBA - 修复：RDP使用BGR顺序，需要转换为RGB
	for y := 0; y < outputHeight && y < inputHeight; y++ {
		for x := 0; x < outputWidth && x < inputWidth; x++ {
			srcIdx := (y*inputWidth + x) * 3
			dstIdx := (y*outputWidth + x) * 4
			// 修复：RDP使用BGR顺序，需要转换为RGB
			b := temp[srcIdx+0] // B (原BGR中的B)
			g := temp[srcIdx+1] // G (原BGR中的G)
			r := temp[srcIdx+2] // R (原BGR中的R)
			// 修复：输出RGBA格式
			output[dstIdx+0] = r   // R
			output[dstIdx+1] = g   // G
			output[dstIdx+2] = b   // B
			output[dstIdx+3] = 255 // A
		}
	}

	return true
}

// BitmapDecompress32 decompresses 32-bit BGRA bitmap data to RGBA format
func BitmapDecompress32(output []byte, outputWidth, outputHeight, inputWidth, inputHeight int, input []byte) bool {
	// Allocate temporary buffer for decompressed 32-bit data
	temp := make([]byte, inputWidth*inputHeight*4)

	// Decompress into temp buffer
	if !bitmapDecompress4(temp, inputWidth, inputHeight, input) {
		return false
	}

	// Convert BGRA to RGBA - 修复：RDP使用BGRA顺序，需要转换为RGBA
	for y := 0; y < outputHeight && y < inputHeight; y++ {
		for x := 0; x < outputWidth && x < inputWidth; x++ {
			srcIdx := (y*inputWidth + x) * 4
			dstIdx := (y*outputWidth + x) * 4
			// 修复：RDP使用BGRA顺序，需要转换为RGBA
			b := temp[srcIdx+0] // B (原BGRA中的B)
			g := temp[srcIdx+1] // G (原BGRA中的G)
			r := temp[srcIdx+2] // R (原BGRA中的R)
			a := temp[srcIdx+3] // A (原BGRA中的A)
			// 修复：输出RGBA格式
			output[dstIdx+0] = r // R
			output[dstIdx+1] = g // G
			output[dstIdx+2] = b // B
			output[dstIdx+3] = a // A
		}
	}

	return true
}

// bitmapDecompress2 decompresses 2 bytes per pixel bitmap data
func bitmapDecompress2(output []byte, width, height int, input []byte) bool {
	var (
		lastopcode = -1
		insertmix  = false
		bicolour   = false
		mask       byte
		mixmask    byte
		mix        uint16
		fomMask    byte
		colour1    uint16
		colour2    uint16
		code       byte
		opcode     int
		count      int
		offset     int
		x          = width
		prevLine   []byte
		line       []byte
	)

	inPtr := 0
	for inPtr < len(input) {
		fomMask = 0
		code = input[inPtr]
		inPtr++
		opcode = int(code) >> 4

		// Handle different opcode forms
		switch {
		case opcode >= 0xc && opcode <= 0xe:
			opcode -= 6
			count = int(code) & 0xf
			offset = 16
		case opcode == 0xf:
			opcode = int(code) & 0xf
			if opcode < 9 {
				count = int(input[inPtr])
				inPtr++
				count |= int(input[inPtr]) << 8
				inPtr++
			} else {
				count = 8
				if opcode >= 0xb {
					count = 1
				}
			}
			offset = 0
		default:
			opcode >>= 1
			count = int(code) & 0x1f
			offset = 32
		}

		// Handle special cases for counts
		if offset != 0 {
			isfillormix := (opcode == 2) || (opcode == 7)
			if count == 0 {
				if isfillormix {
					count = int(input[inPtr]) + 1
					inPtr++
				} else {
					count = int(input[inPtr]) + offset
					inPtr++
				}
			} else if isfillormix {
				count <<= 3
			}
		}

		// Process opcodes
		switch opcode {
		case 0: // Fill
			if lastopcode == opcode && !(x == width && prevLine == nil) {
				insertmix = true
			}

		case 8: // Bicolour
			if inPtr+2 > len(input) {
				return false
			}
			colour1 = uint16(input[inPtr]) | uint16(input[inPtr+1])<<8
			inPtr += 2
			fallthrough

		case 3: // Colour
			if inPtr+2 > len(input) {
				return false
			}
			colour2 = uint16(input[inPtr]) | uint16(input[inPtr+1])<<8
			inPtr += 2

		case 6, 7: // SetMix/Mix, SetMix/FillOrMix
			if inPtr+2 > len(input) {
				return false
			}
			mix = uint16(input[inPtr]) | uint16(input[inPtr+1])<<8
			inPtr += 2
			opcode -= 5

		case 9: // FillOrMix_1
			mask = fomMask1
			opcode = 2
			fomMask = fomMask1

		case 0xa: // FillOrMix_2
			mask = fomMask2
			opcode = 2
			fomMask = fomMask2
		}

		lastopcode = opcode
		mixmask = 0

		// Output loop - 修复：实现与C代码REPEAT宏相同的逻辑
		for count > 0 {
			if x >= width {
				if height <= 0 {
					return false
				}
				x = 0
				height--
				prevLine = line
				line = output[height*width*2 : (height+1)*width*2]
			}

			switch opcode {
			case 0: // Fill
				if insertmix {
					if prevLine == nil {
						// 写入mix值（2字节）
						line[x*2] = byte(mix & 0xff)
						line[x*2+1] = byte(mix >> 8)
					} else {
						// 读取前一行值并异或
						prevVal := uint16(prevLine[x*2]) | uint16(prevLine[x*2+1])<<8
						newVal := prevVal ^ mix
						line[x*2] = byte(newVal & 0xff)
						line[x*2+1] = byte(newVal >> 8)
					}
					insertmix = false
					count--
					x++
					continue
				}

				// 修复：实现REPEAT宏的逻辑
				if prevLine == nil {
					// REPEAT(line[x*2] = 0; line[x*2+1] = 0;)
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							line[(x+i)*2] = 0
							line[(x+i)*2+1] = 0
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						line[x*2] = 0
						line[x*2+1] = 0
						count--
						x++
					}
				} else {
					// REPEAT(copy(line[x*2:(x+1)*2], prevLine[x*2:(x+1)*2]))
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							line[(x+i)*2] = prevLine[(x+i)*2]
							line[(x+i)*2+1] = prevLine[(x+i)*2+1]
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						line[x*2] = prevLine[x*2]
						line[x*2+1] = prevLine[x*2+1]
						count--
						x++
					}
				}

			case 1: // Mix
				// 修复：实现REPEAT宏的逻辑
				if prevLine == nil {
					// REPEAT(line[x*2] = mix & 0xff; line[x*2+1] = mix >> 8;)
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							line[(x+i)*2] = byte(mix & 0xff)
							line[(x+i)*2+1] = byte(mix >> 8)
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						line[x*2] = byte(mix & 0xff)
						line[x*2+1] = byte(mix >> 8)
						count--
						x++
					}
				} else {
					// REPEAT(prevVal = prevLine[x*2] | prevLine[x*2+1] << 8; newVal = prevVal ^ mix; line[x*2] = newVal & 0xff; line[x*2+1] = newVal >> 8;)
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							prevVal := uint16(prevLine[(x+i)*2]) | uint16(prevLine[(x+i)*2+1])<<8
							newVal := prevVal ^ mix
							line[(x+i)*2] = byte(newVal & 0xff)
							line[(x+i)*2+1] = byte(newVal >> 8)
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						prevVal := uint16(prevLine[x*2]) | uint16(prevLine[x*2+1])<<8
						newVal := prevVal ^ mix
						line[x*2] = byte(newVal & 0xff)
						line[x*2+1] = byte(newVal >> 8)
						count--
						x++
					}
				}

			case 2: // Fill or Mix
				// 修复：实现REPEAT宏的逻辑
				if prevLine == nil {
					// REPEAT(MASK_UPDATE(); if (mask & mixmask) { line[x*2] = mix & 0xff; line[x*2+1] = mix >> 8; } else { line[x*2] = 0; line[x*2+1] = 0; })
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							mixmask <<= 1
							if mixmask == 0 {
								if fomMask != 0 {
									mask = fomMask
								} else {
									if inPtr >= len(input) {
										return false
									}
									mask = input[inPtr]
									inPtr++
								}
								mixmask = 1
							}
							if mask&mixmask != 0 {
								line[(x+i)*2] = byte(mix & 0xff)
								line[(x+i)*2+1] = byte(mix >> 8)
							} else {
								line[(x+i)*2] = 0
								line[(x+i)*2+1] = 0
							}
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						mixmask <<= 1
						if mixmask == 0 {
							if fomMask != 0 {
								mask = fomMask
							} else {
								if inPtr >= len(input) {
									return false
								}
								mask = input[inPtr]
								inPtr++
							}
							mixmask = 1
						}
						if mask&mixmask != 0 {
							line[x*2] = byte(mix & 0xff)
							line[x*2+1] = byte(mix >> 8)
						} else {
							line[x*2] = 0
							line[x*2+1] = 0
						}
						count--
						x++
					}
				} else {
					// REPEAT(MASK_UPDATE(); if (mask & mixmask) { prevVal = prevLine[x*2] | prevLine[x*2+1] << 8; newVal = prevVal ^ mix; line[x*2] = newVal & 0xff; line[x*2+1] = newVal >> 8; } else { line[x*2] = prevLine[x*2]; line[x*2+1] = prevLine[x*2+1]; })
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							mixmask <<= 1
							if mixmask == 0 {
								if fomMask != 0 {
									mask = fomMask
								} else {
									if inPtr >= len(input) {
										return false
									}
									mask = input[inPtr]
									inPtr++
								}
								mixmask = 1
							}
							if mask&mixmask != 0 {
								prevVal := uint16(prevLine[(x+i)*2]) | uint16(prevLine[(x+i)*2+1])<<8
								newVal := prevVal ^ mix
								line[(x+i)*2] = byte(newVal & 0xff)
								line[(x+i)*2+1] = byte(newVal >> 8)
							} else {
								line[(x+i)*2] = prevLine[(x+i)*2]
								line[(x+i)*2+1] = prevLine[(x+i)*2+1]
							}
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						mixmask <<= 1
						if mixmask == 0 {
							if fomMask != 0 {
								mask = fomMask
							} else {
								if inPtr >= len(input) {
									return false
								}
								mask = input[inPtr]
								inPtr++
							}
							mixmask = 1
						}
						if mask&mixmask != 0 {
							prevVal := uint16(prevLine[x*2]) | uint16(prevLine[x*2+1])<<8
							newVal := prevVal ^ mix
							line[x*2] = byte(newVal & 0xff)
							line[x*2+1] = byte(newVal >> 8)
						} else {
							line[x*2] = prevLine[x*2]
							line[x*2+1] = prevLine[x*2+1]
						}
						count--
						x++
					}
				}

			case 3: // Colour
				// 修复：实现REPEAT宏的逻辑
				// REPEAT(line[x*2] = colour2 & 0xff; line[x*2+1] = colour2 >> 8;)
				for (count&^0x7) != 0 && (x+8) < width {
					for i := 0; i < 8; i++ {
						line[(x+i)*2] = byte(colour2 & 0xff)
						line[(x+i)*2+1] = byte(colour2 >> 8)
					}
					count -= 8
					x += 8
				}
				for count > 0 && x < width {
					line[x*2] = byte(colour2 & 0xff)
					line[x*2+1] = byte(colour2 >> 8)
					count--
					x++
				}

			case 4: // Copy
				// 修复：实现REPEAT宏的逻辑
				// REPEAT(line[x*2] = input[inPtr]; line[x*2+1] = input[inPtr+1]; inPtr += 2;)
				for (count&^0x7) != 0 && (x+8) < width {
					for i := 0; i < 8; i++ {
						if inPtr+2 > len(input) {
							return false
						}
						line[(x+i)*2] = input[inPtr]
						line[(x+i)*2+1] = input[inPtr+1]
						inPtr += 2
					}
					count -= 8
					x += 8
				}
				for count > 0 && x < width {
					if inPtr+2 > len(input) {
						return false
					}
					line[x*2] = input[inPtr]
					line[x*2+1] = input[inPtr+1]
					inPtr += 2
					count--
					x++
				}

			case 8: // Bicolour
				// 修复：实现REPEAT宏的逻辑
				// REPEAT(if (bicolour) { line[x*2] = colour2 & 0xff; line[x*2+1] = colour2 >> 8; bicolour = false; } else { line[x*2] = colour1 & 0xff; line[x*2+1] = colour1 >> 8; bicolour = true; count++; })
				for (count&^0x7) != 0 && (x+8) < width {
					for i := 0; i < 8; i++ {
						if bicolour {
							line[(x+i)*2] = byte(colour2 & 0xff)
							line[(x+i)*2+1] = byte(colour2 >> 8)
							bicolour = false
						} else {
							line[(x+i)*2] = byte(colour1 & 0xff)
							line[(x+i)*2+1] = byte(colour1 >> 8)
							bicolour = true
							count++
						}
					}
					count -= 8
					x += 8
				}
				for count > 0 && x < width {
					if bicolour {
						line[x*2] = byte(colour2 & 0xff)
						line[x*2+1] = byte(colour2 >> 8)
						bicolour = false
					} else {
						line[x*2] = byte(colour1 & 0xff)
						line[x*2+1] = byte(colour1 >> 8)
						bicolour = true
						count++
					}
					count--
					x++
				}

			case 0xd: // White
				// 修复：实现REPEAT宏的逻辑
				// REPEAT(line[x*2] = 0xff; line[x*2+1] = 0xff;)
				for (count&^0x7) != 0 && (x+8) < width {
					for i := 0; i < 8; i++ {
						line[(x+i)*2] = 0xff
						line[(x+i)*2+1] = 0xff
					}
					count -= 8
					x += 8
				}
				for count > 0 && x < width {
					line[x*2] = 0xff
					line[x*2+1] = 0xff
					count--
					x++
				}

			case 0xe: // Black
				// 修复：实现REPEAT宏的逻辑
				// REPEAT(line[x*2] = 0; line[x*2+1] = 0;)
				for (count&^0x7) != 0 && (x+8) < width {
					for i := 0; i < 8; i++ {
						line[(x+i)*2] = 0
						line[(x+i)*2+1] = 0
					}
					count -= 8
					x += 8
				}
				for count > 0 && x < width {
					line[x*2] = 0
					line[x*2+1] = 0
					count--
					x++
				}

			default:
				return false
			}
		}
	}

	return true
}

// bitmapDecompress3 decompresses 3 bytes per pixel bitmap data
func bitmapDecompress3(output []byte, width, height int, input []byte) bool {
	var (
		lastopcode = -1
		insertmix  = false
		bicolour   = false
		mask       byte
		mixmask    byte
		mix        = [3]byte{0xff, 0xff, 0xff}
		fomMask    byte
		colour1    = [3]byte{0, 0, 0}
		colour2    = [3]byte{0, 0, 0}
		code       byte
		opcode     int
		count      int
		offset     int
		x          = width
		prevLine   []byte
		line       []byte
	)

	inPtr := 0
	for inPtr < len(input) {
		fomMask = 0
		code = input[inPtr]
		inPtr++
		opcode = int(code) >> 4

		// Handle different opcode forms
		switch {
		case opcode >= 0xc && opcode <= 0xe:
			opcode -= 6
			count = int(code) & 0xf
			offset = 16
		case opcode == 0xf:
			opcode = int(code) & 0xf
			if opcode < 9 {
				count = int(input[inPtr])
				inPtr++
				count |= int(input[inPtr]) << 8
				inPtr++
			} else {
				count = 8
				if opcode >= 0xb {
					count = 1
				}
			}
			offset = 0
		default:
			opcode >>= 1
			count = int(code) & 0x1f
			offset = 32
		}

		// Handle special cases for counts
		if offset != 0 {
			isfillormix := (opcode == 2) || (opcode == 7)
			if count == 0 {
				if isfillormix {
					count = int(input[inPtr]) + 1
					inPtr++
				} else {
					count = int(input[inPtr]) + offset
					inPtr++
				}
			} else if isfillormix {
				count <<= 3
			}
		}

		// Process opcodes
		switch opcode {
		case 0: // Fill
			if lastopcode == opcode && !(x == width && prevLine == nil) {
				insertmix = true
			}

		case 8: // Bicolour
			if inPtr+3 > len(input) {
				return false
			}
			copy(colour1[:], input[inPtr:inPtr+3])
			inPtr += 3
			fallthrough

		case 3: // Colour
			if inPtr+3 > len(input) {
				return false
			}
			copy(colour2[:], input[inPtr:inPtr+3])
			inPtr += 3

		case 6, 7: // SetMix/Mix, SetMix/FillOrMix
			if inPtr+3 > len(input) {
				return false
			}
			copy(mix[:], input[inPtr:inPtr+3])
			inPtr += 3
			opcode -= 5

		case 9: // FillOrMix_1
			mask = fomMask1
			opcode = 2
			fomMask = fomMask1

		case 0xa: // FillOrMix_2
			mask = fomMask2
			opcode = 2
			fomMask = fomMask2
		}

		lastopcode = opcode
		mixmask = 0

		// Output loop - 修复：实现与C代码REPEAT宏相同的逻辑
		for count > 0 {
			if x >= width {
				if height <= 0 {
					return false
				}
				x = 0
				height--
				prevLine = output[(height+1)*width*3 : (height+2)*width*3]
				line = output[height*width*3 : (height+1)*width*3]
			}

			switch opcode {
			case 0: // Fill
				if insertmix {
					if prevLine == nil {
						copy(line[x*3:(x+1)*3], mix[:])
					} else {
						for i := 0; i < 3; i++ {
							line[x*3+i] = prevLine[x*3+i] ^ mix[i]
						}
					}
					insertmix = false
					count--
					x++
					continue
				}

				// 修复：实现REPEAT宏的逻辑
				if prevLine == nil {
					// REPEAT(line[x*3] = 0; line[x*3+1] = 0; line[x*3+2] = 0;)
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							line[(x+i)*3] = 0
							line[(x+i)*3+1] = 0
							line[(x+i)*3+2] = 0
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						line[x*3] = 0
						line[x*3+1] = 0
						line[x*3+2] = 0
						count--
						x++
					}
				} else {
					// REPEAT(copy(line[x*3:(x+1)*3], prevLine[x*3:(x+1)*3]))
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							copy(line[(x+i)*3:(x+i+1)*3], prevLine[(x+i)*3:(x+i+1)*3])
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						copy(line[x*3:(x+1)*3], prevLine[x*3:(x+1)*3])
						count--
						x++
					}
				}

			case 1: // Mix
				// 修复：实现REPEAT宏的逻辑
				if prevLine == nil {
					// REPEAT(copy(line[x*3:(x+1)*3], mix[:]))
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							copy(line[(x+i)*3:(x+i+1)*3], mix[:])
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						copy(line[x*3:(x+1)*3], mix[:])
						count--
						x++
					}
				} else {
					// REPEAT(for j := 0; j < 3; j++ { line[x*3+j] = prevLine[x*3+j] ^ mix[j]; })
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							for j := 0; j < 3; j++ {
								line[(x+i)*3+j] = prevLine[(x+i)*3+j] ^ mix[j]
							}
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						for j := 0; j < 3; j++ {
							line[x*3+j] = prevLine[x*3+j] ^ mix[j]
						}
						count--
						x++
					}
				}

			case 2: // Fill or Mix
				// 修复：实现REPEAT宏的逻辑
				if prevLine == nil {
					// REPEAT(MASK_UPDATE(); if (mask & mixmask) copy(line[x*3:(x+1)*3], mix[:]); else { line[x*3] = 0; line[x*3+1] = 0; line[x*3+2] = 0; })
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							mixmask <<= 1
							if mixmask == 0 {
								if fomMask != 0 {
									mask = fomMask
								} else {
									if inPtr >= len(input) {
										return false
									}
									mask = input[inPtr]
									inPtr++
								}
								mixmask = 1
							}
							if mask&mixmask != 0 {
								copy(line[(x+i)*3:(x+i+1)*3], mix[:])
							} else {
								line[(x+i)*3] = 0
								line[(x+i)*3+1] = 0
								line[(x+i)*3+2] = 0
							}
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						mixmask <<= 1
						if mixmask == 0 {
							if fomMask != 0 {
								mask = fomMask
							} else {
								if inPtr >= len(input) {
									return false
								}
								mask = input[inPtr]
								inPtr++
							}
							mixmask = 1
						}
						if mask&mixmask != 0 {
							copy(line[x*3:(x+1)*3], mix[:])
						} else {
							line[x*3] = 0
							line[x*3+1] = 0
							line[x*3+2] = 0
						}
						count--
						x++
					}
				} else {
					// REPEAT(MASK_UPDATE(); if (mask & mixmask) for j := 0; j < 3; j++ { line[x*3+j] = prevLine[x*3+j] ^ mix[j]; } else copy(line[x*3:(x+1)*3], prevLine[x*3:(x+1)*3]);)
					for (count&^0x7) != 0 && (x+8) < width {
						for i := 0; i < 8; i++ {
							mixmask <<= 1
							if mixmask == 0 {
								if fomMask != 0 {
									mask = fomMask
								} else {
									if inPtr >= len(input) {
										return false
									}
									mask = input[inPtr]
									inPtr++
								}
								mixmask = 1
							}
							if mask&mixmask != 0 {
								for j := 0; j < 3; j++ {
									line[(x+i)*3+j] = prevLine[(x+i)*3+j] ^ mix[j]
								}
							} else {
								copy(line[(x+i)*3:(x+i+1)*3], prevLine[(x+i)*3:(x+i+1)*3])
							}
						}
						count -= 8
						x += 8
					}
					for count > 0 && x < width {
						mixmask <<= 1
						if mixmask == 0 {
							if fomMask != 0 {
								mask = fomMask
							} else {
								if inPtr >= len(input) {
									return false
								}
								mask = input[inPtr]
								inPtr++
							}
							mixmask = 1
						}
						if mask&mixmask != 0 {
							for j := 0; j < 3; j++ {
								line[x*3+j] = prevLine[x*3+j] ^ mix[j]
							}
						} else {
							copy(line[x*3:(x+1)*3], prevLine[x*3:(x+1)*3])
						}
						count--
						x++
					}
				}

			case 3: // Colour
				// 修复：实现REPEAT宏的逻辑
				// REPEAT(copy(line[x*3:(x+1)*3], colour2[:]))
				for (count&^0x7) != 0 && (x+8) < width {
					for i := 0; i < 8; i++ {
						copy(line[(x+i)*3:(x+i+1)*3], colour2[:])
					}
					count -= 8
					x += 8
				}
				for count > 0 && x < width {
					copy(line[x*3:(x+1)*3], colour2[:])
					count--
					x++
				}

			case 4: // Copy
				// 修复：实现REPEAT宏的逻辑
				// REPEAT(copy(line[x*3:(x+1)*3], input[inPtr:inPtr+3]); inPtr += 3;)
				for (count&^0x7) != 0 && (x+8) < width {
					for i := 0; i < 8; i++ {
						if inPtr+3 > len(input) {
							return false
						}
						copy(line[(x+i)*3:(x+i+1)*3], input[inPtr:inPtr+3])
						inPtr += 3
					}
					count -= 8
					x += 8
				}
				for count > 0 && x < width {
					if inPtr+3 > len(input) {
						return false
					}
					copy(line[x*3:(x+1)*3], input[inPtr:inPtr+3])
					inPtr += 3
					count--
					x++
				}

			case 8: // Bicolour
				// 修复：实现REPEAT宏的逻辑
				// REPEAT(if (bicolour) { copy(line[x*3:(x+1)*3], colour2[:]); bicolour = false; } else { copy(line[x*3:(x+1)*3], colour1[:]); bicolour = true; count++; })
				for (count&^0x7) != 0 && (x+8) < width {
					for i := 0; i < 8; i++ {
						if bicolour {
							copy(line[(x+i)*3:(x+i+1)*3], colour2[:])
							bicolour = false
						} else {
							copy(line[(x+i)*3:(x+i+1)*3], colour1[:])
							bicolour = true
							count++
						}
					}
					count -= 8
					x += 8
				}
				for count > 0 && x < width {
					if bicolour {
						copy(line[x*3:(x+1)*3], colour2[:])
						bicolour = false
					} else {
						copy(line[x*3:(x+1)*3], colour1[:])
						bicolour = true
						count++
					}
					count--
					x++
				}

			case 0xd: // White
				// 修复：实现REPEAT宏的逻辑
				// REPEAT(line[x*3] = 0xff; line[x*3+1] = 0xff; line[x*3+2] = 0xff;)
				for (count&^0x7) != 0 && (x+8) < width {
					for i := 0; i < 8; i++ {
						line[(x+i)*3] = 0xff
						line[(x+i)*3+1] = 0xff
						line[(x+i)*3+2] = 0xff
					}
					count -= 8
					x += 8
				}
				for count > 0 && x < width {
					line[x*3] = 0xff
					line[x*3+1] = 0xff
					line[x*3+2] = 0xff
					count--
					x++
				}

			case 0xe: // Black
				// 修复：实现REPEAT宏的逻辑
				// REPEAT(line[x*3] = 0; line[x*3+1] = 0; line[x*3+2] = 0;)
				for (count&^0x7) != 0 && (x+8) < width {
					for i := 0; i < 8; i++ {
						line[(x+i)*3] = 0
						line[(x+i)*3+1] = 0
						line[(x+i)*3+2] = 0
					}
					count -= 8
					x += 8
				}
				for count > 0 && x < width {
					line[x*3] = 0
					line[x*3+1] = 0
					line[x*3+2] = 0
					count--
					x++
				}

			default:
				return false
			}
		}
	}

	return true
}

// bitmapDecompress4 decompresses 4 bytes per pixel bitmap data
func bitmapDecompress4(output []byte, width, height int, input []byte) bool {
	if len(input) == 0 {
		return false
	}

	if input[0] != 0x10 {
		return false
	}

	totalProcessed := 1
	inPtr := 1

	// 修复：按照C代码的顺序处理颜色平面 (BGRA -> RGBA)
	// 处理B平面 (output + 2)
	bytesProcessed := processPlane(input[inPtr:], width, height, output[2:], len(input)-totalProcessed)
	if bytesProcessed <= 0 {
		return false
	}
	totalProcessed += bytesProcessed
	inPtr += bytesProcessed

	// 处理G平面 (output + 1)
	bytesProcessed = processPlane(input[inPtr:], width, height, output[1:], len(input)-totalProcessed)
	if bytesProcessed <= 0 {
		return false
	}
	totalProcessed += bytesProcessed
	inPtr += bytesProcessed

	// 处理R平面 (output + 0)
	bytesProcessed = processPlane(input[inPtr:], width, height, output[0:], len(input)-totalProcessed)
	if bytesProcessed <= 0 {
		return false
	}
	totalProcessed += bytesProcessed
	inPtr += bytesProcessed

	// 处理A平面 (output + 3)
	bytesProcessed = processPlane(input[inPtr:], width, height, output[3:], len(input)-totalProcessed)
	if bytesProcessed <= 0 {
		return false
	}
	totalProcessed += bytesProcessed

	return totalProcessed == len(input)
}

// processPlane decompresses a single color plane
func processPlane(input []byte, width, height int, output []byte, size int) int {
	var (
		inPtr    int
		lastLine []byte
		thisLine []byte
		x        int
		code     byte
		collen   int
		replen   int
		color    int
		revcode  int
	)

	for indexh := 0; indexh < height; indexh++ {
		thisLine = output[(height-indexh-1)*width*4:]
		x = 0
		color = 0

		if lastLine == nil {
			for x < width {
				if inPtr >= len(input) {
					return -1
				}

				code = input[inPtr]
				inPtr++

				replen = int(code & 0xf)
				collen = int((code >> 4) & 0xf)
				revcode = (replen << 4) | collen

				if revcode <= 47 && revcode >= 16 {
					replen = revcode
					collen = 0
				}

				for collen > 0 {
					if inPtr >= len(input) {
						return -1
					}
					color = int(input[inPtr])
					inPtr++
					thisLine[x*4] = byte(color)
					x++
					collen--
				}

				for replen > 0 {
					thisLine[x*4] = byte(color)
					x++
					replen--
				}
			}
		} else {
			for x < width {
				if inPtr >= len(input) {
					return -1
				}

				code = input[inPtr]
				inPtr++

				replen = int(code & 0xf)
				collen = int((code >> 4) & 0xf)
				revcode = (replen << 4) | collen

				if revcode <= 47 && revcode >= 16 {
					replen = revcode
					collen = 0
				}

				for collen > 0 {
					if inPtr >= len(input) {
						return -1
					}
					x2 := int(input[inPtr])
					inPtr++

					if x2&1 != 0 {
						x2 = (x2 >> 1) + 1
						color = -x2
					} else {
						x2 = x2 >> 1
						color = x2
					}

					x2 = int(lastLine[x*4]) + color
					thisLine[x*4] = byte(x2)
					x++
					collen--
				}

				for replen > 0 {
					x2 := int(lastLine[x*4]) + color
					thisLine[x*4] = byte(x2)
					x++
					replen--
				}
			}
		}

		lastLine = thisLine
	}

	return inPtr
}

// DebugColorConversion 调试颜色转换函数
func DebugColorConversion() {
	fmt.Println("=== 颜色转换调试信息 ===")

	// 测试15位颜色转换
	test15Bit := uint16(0x7C1F) // RGB555: 红色最大值
	r15 := uint8((test15Bit & 0x7c00) >> 10)
	g15 := uint8((test15Bit & 0x03e0) >> 5)
	b15 := uint8(test15Bit & 0x001f)
	r15_8 := r15 * 255 / 31
	g15_8 := g15 * 255 / 31
	b15_8 := b15 * 255 / 31

	fmt.Printf("15位测试 (0x%04X): R=%d(%d) G=%d(%d) B=%d(%d)\n",
		test15Bit, r15, r15_8, g15, g15_8, b15, b15_8)

	// 测试16位颜色转换
	test16Bit := uint16(0xF81F) // RGB565: 红色最大值
	r16 := uint8((test16Bit & 0xf800) >> 11)
	g16 := uint8((test16Bit & 0x07e0) >> 5)
	b16 := uint8(test16Bit & 0x001f)
	r16_8 := r16 * 255 / 31
	g16_8 := g16 * 255 / 63
	b16_8 := b16 * 255 / 31

	fmt.Printf("16位测试 (0x%04X): R=%d(%d) G=%d(%d) B=%d(%d)\n",
		test16Bit, r16, r16_8, g16, g16_8, b16, b16_8)

	// 测试白色
	white15 := uint16(0x7FFF) // RGB555白色
	white16 := uint16(0xFFFF) // RGB565白色

	r15w := uint8((white15&0x7c00)>>10) * 255 / 31
	g15w := uint8((white15&0x03e0)>>5) * 255 / 31
	b15w := uint8(white15&0x001f) * 255 / 31

	r16w := uint8((white16&0xf800)>>11) * 255 / 31
	g16w := uint8((white16&0x07e0)>>5) * 255 / 63
	b16w := uint8(white16&0x001f) * 255 / 31

	fmt.Printf("15位白色 (0x%04X): R=%d G=%d B=%d\n", white15, r15w, g15w, b15w)
	fmt.Printf("16位白色 (0x%04X): R=%d G=%d B=%d\n", white16, r16w, g16w, b16w)

	fmt.Println("=== 调试信息结束 ===")
}
