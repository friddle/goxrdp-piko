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

			// 修复：使用与core/io.go中RGB555ToRGB函数完全相同的转换方式
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

			// 修复：使用与core/io.go中RGB565ToRGB函数完全相同的转换方式
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
		prevline, line, count            int
		offset, code                     int
		x                                int = width
		opcode                           int
		lastopcode                       int = -1
		insertmix, bicolour, isfillormix bool
		mixmask, mask                    byte
		colour1, colour2                 uint16
		mix                              uint16 = 0xffff
		fom_mask                         byte
	)

	out := make([]uint16, width*height)
	for len(input) != 0 {
		fom_mask = 0
		code = int(input[0])
		input = input[1:]
		opcode = code >> 4

		// Handle different opcode forms
		switch opcode {
		case 0xc, 0xd, 0xe:
			opcode -= 6
			count = code & 0xf
			offset = 16
		case 0xf:
			opcode = code & 0xf
			if opcode < 9 {
				count = int(input[0])
				input = input[1:]
				count |= int(input[0]) << 8
				input = input[1:]
			} else {
				count = 1
				if opcode < 0xb {
					count = 8
				}
			}
			offset = 0
		default:
			opcode >>= 1
			count = code & 0x1f
			offset = 32
		}

		// Handle strange cases for counts
		if offset != 0 {
			isfillormix = ((opcode == 2) || (opcode == 7))
			if count == 0 {
				if isfillormix {
					count = int(input[0]) + 1
					input = input[1:]
				} else {
					count = int(input[0]) + offset
					input = input[1:]
				}
			} else if isfillormix {
				count <<= 3
			}
		}

		// Read preliminary data
		switch opcode {
		case 0: // Fill
			if (lastopcode == opcode) && !((x == width) && (prevline == 0)) {
				insertmix = true
			}
		case 8: // Bicolour
			// 修复：使用正确的字节序读取16位值
			colour1 = uint16(input[0]) | uint16(input[1])<<8
			input = input[2:]
			fallthrough
		case 3: // Colour
			// 修复：使用正确的字节序读取16位值
			colour2 = uint16(input[0]) | uint16(input[1])<<8
			input = input[2:]
		case 6, 7: // SetMix/Mix, SetMix/FillOrMix
			// 修复：使用正确的字节序读取16位值
			mix = uint16(input[0]) | uint16(input[1])<<8
			input = input[2:]
			opcode -= 5
		case 9: // FillOrMix_1
			mask = 0x03
			opcode = 0x02
			fom_mask = 3
		case 0x0a: // FillOrMix_2
			mask = 0x05
			opcode = 0x02
			fom_mask = 5
		}

		lastopcode = opcode
		mixmask = 0

		// Output body
		for count > 0 {
			if x >= width {
				if height <= 0 {
					return false
				}
				x = 0
				height--
				prevline = line
				line = height * width
			}

			switch opcode {
			case 0: // Fill
				if insertmix {
					if prevline == 0 {
						out[x+line] = mix
					} else {
						out[x+line] = out[prevline+x] ^ mix
					}
					insertmix = false
					count--
					x++
				}
				if prevline == 0 {
					REPEAT(func() {
						out[x+line] = 0
					}, &count, &x, width)
				} else {
					REPEAT(func() {
						out[x+line] = out[prevline+x]
					}, &count, &x, width)
				}
			case 1: // Mix
				if prevline == 0 {
					REPEAT(func() {
						out[x+line] = mix
					}, &count, &x, width)
				} else {
					REPEAT(func() {
						out[x+line] = out[prevline+x] ^ mix
					}, &count, &x, width)
				}
			case 2: // Fill or Mix
				if prevline == 0 {
					REPEAT(func() {
						mixmask <<= 1
						if mixmask == 0 {
							mask = fom_mask
							if fom_mask == 0 {
								mask = input[0]
								input = input[1:]
								mixmask = 1
							}
						}
						if mask&mixmask != 0 {
							out[x+line] = mix
						} else {
							out[x+line] = 0
						}
					}, &count, &x, width)
				} else {
					REPEAT(func() {
						mixmask <<= 1
						if mixmask == 0 {
							mask = fom_mask
							if fom_mask == 0 {
								mask = input[0]
								input = input[1:]
								mixmask = 1
							}
						}
						if mask&mixmask != 0 {
							out[x+line] = out[prevline+x] ^ mix
						} else {
							out[x+line] = out[prevline+x]
						}
					}, &count, &x, width)
				}
			case 3: // Colour
				REPEAT(func() {
					out[x+line] = colour2
				}, &count, &x, width)
			case 4: // Copy
				REPEAT(func() {
					// 修复：使用正确的字节序读取16位值
					var val uint16
					val = uint16(input[0]) | uint16(input[1])<<8
					input = input[2:]
					out[x+line] = val
				}, &count, &x, width)
			case 8: // Bicolour
				REPEAT(func() {
					if bicolour {
						out[x+line] = colour2
						bicolour = false
					} else {
						out[x+line] = colour1
						bicolour = true
						count++
					}
				}, &count, &x, width)
			case 0xd: // White
				REPEAT(func() {
					out[x+line] = 0xffff
				}, &count, &x, width)
			case 0xe: // Black
				REPEAT(func() {
					out[x+line] = 0
				}, &count, &x, width)
			default:
				fmt.Printf("bitmap opcode 0x%x\n", opcode)
				return false
			}
		}
	}

	// 修复：将uint16数组转换为字节数组，使用正确的字节序
	j := 0
	for _, v := range out {
		output[j] = byte(v & 0xff)
		output[j+1] = byte(v >> 8)
		j += 2
	}

	return true
}

// REPEAT macro implementation
func REPEAT(f func(), count *int, x *int, width int) {
	for (*count & ^0x7) != 0 && ((*x + 8) < width) {
		for i := 0; i < 8; i++ {
			f()
			*count = *count - 1
			*x = *x + 1
		}
	}

	for (*count > 0) && (*x < width) {
		f()
		*count = *count - 1
		*x = *x + 1
	}
}

// bitmapDecompress3 decompresses 3 bytes per pixel bitmap data
func bitmapDecompress3(output []byte, width, height int, input []byte) bool {
	var (
		prevline, line, count            int
		opcode, offset, code             int
		x                                int = width
		lastopcode                       int = -1
		insertmix, bicolour, isfillormix bool
		mixmask, mask                    byte
		colour1                          = [3]byte{0, 0, 0}
		colour2                          = [3]byte{0, 0, 0}
		mix                              = [3]byte{0xff, 0xff, 0xff}
		fom_mask                         byte
	)

	out := output
	for len(input) != 0 {
		fom_mask = 0
		code = int(input[0])
		input = input[1:]
		opcode = code >> 4

		// Handle different opcode forms
		switch opcode {
		case 0xc, 0xd, 0xe:
			opcode -= 6
			count = code & 0xf
			offset = 16
		case 0xf:
			opcode = code & 0xf
			if opcode < 9 {
				count = int(input[0])
				input = input[1:]
				count |= int(input[0]) << 8
				input = input[1:]
			} else {
				count = 1
				if opcode < 0xb {
					count = 8
				}
			}
			offset = 0
		default:
			opcode >>= 1
			count = code & 0x1f
			offset = 32
		}

		// Handle strange cases for counts
		if offset != 0 {
			isfillormix = ((opcode == 2) || (opcode == 7))
			if count == 0 {
				if isfillormix {
					count = int(input[0]) + 1
					input = input[1:]
				} else {
					count = int(input[0]) + offset
					input = input[1:]
				}
			} else if isfillormix {
				count <<= 3
			}
		}

		// Read preliminary data
		switch opcode {
		case 0: // Fill
			if (lastopcode == opcode) && !((x == width) && (prevline == 0)) {
				insertmix = true
			}
		case 8: // Bicolour
			colour1[0] = input[0]
			colour1[1] = input[1]
			colour1[2] = input[2]
			input = input[3:]
			fallthrough
		case 3: // Colour
			colour2[0] = input[0]
			colour2[1] = input[1]
			colour2[2] = input[2]
			input = input[3:]
		case 6, 7: // SetMix/Mix, SetMix/FillOrMix
			mix[0] = input[0]
			mix[1] = input[1]
			mix[2] = input[2]
			input = input[3:]
			opcode -= 5
		case 9: // FillOrMix_1
			mask = 0x03
			opcode = 0x02
			fom_mask = 3
		case 0x0a: // FillOrMix_2
			mask = 0x05
			opcode = 0x02
			fom_mask = 5
		}

		lastopcode = opcode
		mixmask = 0

		// Output body
		for count > 0 {
			if x >= width {
				if height <= 0 {
					return false
				}
				x = 0
				height--
				prevline = line
				line = height * width * 3
			}

			switch opcode {
			case 0: // Fill
				if insertmix {
					if prevline == 0 {
						out[3*x+line] = mix[0]
						out[3*x+line+1] = mix[1]
						out[3*x+line+2] = mix[2]
					} else {
						out[3*x+line] = out[prevline+3*x] ^ mix[0]
						out[3*x+line+1] = out[prevline+3*x+1] ^ mix[1]
						out[3*x+line+2] = out[prevline+3*x+2] ^ mix[2]
					}
					insertmix = false
					count--
					x++
				}
				if prevline == 0 {
					REPEAT(func() {
						out[3*x+line] = 0
						out[3*x+line+1] = 0
						out[3*x+line+2] = 0
					}, &count, &x, width)
				} else {
					REPEAT(func() {
						out[3*x+line] = out[prevline+3*x]
						out[3*x+line+1] = out[prevline+3*x+1]
						out[3*x+line+2] = out[prevline+3*x+2]
					}, &count, &x, width)
				}
			case 1: // Mix
				if prevline == 0 {
					REPEAT(func() {
						out[3*x+line] = mix[0]
						out[3*x+line+1] = mix[1]
						out[3*x+line+2] = mix[2]
					}, &count, &x, width)
				} else {
					REPEAT(func() {
						out[3*x+line] = out[prevline+3*x] ^ mix[0]
						out[3*x+line+1] = out[prevline+3*x+1] ^ mix[1]
						out[3*x+line+2] = out[prevline+3*x+2] ^ mix[2]
					}, &count, &x, width)
				}
			case 2: // Fill or Mix
				if prevline == 0 {
					REPEAT(func() {
						mixmask <<= 1
						if mixmask == 0 {
							mask = fom_mask
							if fom_mask == 0 {
								mask = input[0]
								input = input[1:]
								mixmask = 1
							}
						}
						if mask&mixmask != 0 {
							out[3*x+line] = mix[0]
							out[3*x+line+1] = mix[1]
							out[3*x+line+2] = mix[2]
						} else {
							out[3*x+line] = 0
							out[3*x+line+1] = 0
							out[3*x+line+2] = 0
						}
					}, &count, &x, width)
				} else {
					REPEAT(func() {
						mixmask <<= 1
						if mixmask == 0 {
							mask = fom_mask
							if fom_mask == 0 {
								mask = input[0]
								input = input[1:]
								mixmask = 1
							}
						}
						if mask&mixmask != 0 {
							out[3*x+line] = out[prevline+3*x] ^ mix[0]
							out[3*x+line+1] = out[prevline+3*x+1] ^ mix[1]
							out[3*x+line+2] = out[prevline+3*x+2] ^ mix[2]
						} else {
							out[3*x+line] = out[prevline+3*x]
							out[3*x+line+1] = out[prevline+3*x+1]
							out[3*x+line+2] = out[prevline+3*x+2]
						}
					}, &count, &x, width)
				}
			case 3: // Colour
				REPEAT(func() {
					out[3*x+line] = colour2[0]
					out[3*x+line+1] = colour2[1]
					out[3*x+line+2] = colour2[2]
				}, &count, &x, width)
			case 4: // Copy
				REPEAT(func() {
					out[3*x+line] = input[0]
					out[3*x+line+1] = input[1]
					out[3*x+line+2] = input[2]
					input = input[3:]
				}, &count, &x, width)
			case 8: // Bicolour
				REPEAT(func() {
					if bicolour {
						out[3*x+line] = colour2[0]
						out[3*x+line+1] = colour2[1]
						out[3*x+line+2] = colour2[2]
						bicolour = false
					} else {
						out[3*x+line] = colour1[0]
						out[3*x+line+1] = colour1[1]
						out[3*x+line+2] = colour1[2]
						bicolour = true
						count++
					}
				}, &count, &x, width)
			case 0xd: // White
				REPEAT(func() {
					out[3*x+line] = 0xff
					out[3*x+line+1] = 0xff
					out[3*x+line+2] = 0xff
				}, &count, &x, width)
			case 0xe: // Black
				REPEAT(func() {
					out[3*x+line] = 0
					out[3*x+line+1] = 0
					out[3*x+line+2] = 0
				}, &count, &x, width)
			default:
				fmt.Printf("bitmap opcode 0x%x\n", opcode)
				return false
			}
		}
	}

	return true
}

// bitmapDecompress4 decompresses 4 bytes per pixel bitmap data
func bitmapDecompress4(output []byte, width, height int, input []byte) bool {
	var (
		code             int
		onceBytes, total int
	)

	code = int(input[0])
	input = input[1:]
	if code != 0x10 {
		return false
	}

	total = 1
	onceBytes = processPlane(input, width, height, output, 3)
	total += onceBytes
	input = input[onceBytes:]

	onceBytes = processPlane(input, width, height, output, 2)
	total += onceBytes
	input = input[onceBytes:]

	onceBytes = processPlane(input, width, height, output, 1)
	total += onceBytes
	input = input[onceBytes:]

	onceBytes = processPlane(input, width, height, output, 0)
	total += onceBytes

	return total == len(input)+1 // +1 for the initial code byte
}

// processPlane decompresses a single color plane
func processPlane(input []byte, width, height int, output []byte, offset int) int {
	var (
		indexw   int
		indexh   int
		code     int
		collen   int
		replen   int
		color    byte
		x        byte
		revcode  int
		lastline int
		thisline int
	)

	ln := len(input)
	lastline = 0
	indexh = 0
	i := 0

	for indexh < height {
		thisline = offset + (width * height * 4) - ((indexh + 1) * width * 4)
		color = 0
		indexw = 0
		i = thisline

		if lastline == 0 {
			for indexw < width {
				code = int(input[0])
				input = input[1:]
				replen = code & 0xf
				collen = (code >> 4) & 0xf
				revcode = (replen << 4) | collen
				if (revcode <= 47) && (revcode >= 16) {
					replen = revcode
					collen = 0
				}
				for collen > 0 {
					color = input[0]
					input = input[1:]
					output[i] = color
					i += 4
					indexw++
					collen--
				}
				for replen > 0 {
					output[i] = color
					i += 4
					indexw++
					replen--
				}
			}
		} else {
			for indexw < width {
				code = int(input[0])
				input = input[1:]
				replen = code & 0xf
				collen = (code >> 4) & 0xf
				revcode = (replen << 4) | collen
				if (revcode <= 47) && (revcode >= 16) {
					replen = revcode
					collen = 0
				}
				for collen > 0 {
					x = input[0]
					input = input[1:]
					if x&1 != 0 {
						x = x >> 1
						x = x + 1
						color = -x
					} else {
						x = x >> 1
						color = x
					}
					x = output[indexw*4+lastline] + color
					output[i] = x
					i += 4
					indexw++
					collen--
				}
				for replen > 0 {
					x = output[indexw*4+lastline] + color
					output[i] = x
					i += 4
					indexw++
					replen--
				}
			}
		}
		indexh++
		lastline = thisline
	}
	return ln - len(input)
}

// DebugColorConversion 调试颜色转换函数
func DebugColorConversion() {
	fmt.Println("=== RLE解压缩函数调试信息 ===")
	fmt.Println("BitmapDecompress15: 15位RGB555解压缩")
	fmt.Println("BitmapDecompress16: 16位RGB565解压缩")
	fmt.Println("BitmapDecompress24: 24位BGR解压缩")
	fmt.Println("BitmapDecompress32: 32位BGRA解压缩")
	fmt.Println("REPEAT宏: 已正确实现")
	fmt.Println("字节序: 使用小端序（低字节在前）")
	fmt.Println("颜色格式: RDP BGR/BGRA -> RGBA")
}
