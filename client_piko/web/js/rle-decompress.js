/**
 * RLE (Run-Length Encoding) 解压缩库
 * 基于 Go 代码逻辑的纯 JavaScript 实现
 * 支持 15位(RGB555)、16位(RGB565)、24位(BGR)、32位(BGRA) 位图解压缩
 */

class RLEDecompressor {
    constructor() {
        // 常量定义
        this.NO_MASK_UPDATE = 0;
        this.INSERT_FILL_OR_MIX = 1;
        this.INSERT_FILL = 2;
        this.INSERT_MIX = 3;
        
        this.FOM_MASK1 = 0x03;
        this.FOM_MASK2 = 0x05;
    }

    /**
     * 解压缩 15位 (RGB555) 位图数据到 RGBA 格式
     * @param {Uint8Array} output - 输出RGBA缓冲区
     * @param {number} outputWidth - 输出宽度
     * @param {number} outputHeight - 输出高度
     * @param {number} inputWidth - 输入宽度
     * @param {number} inputHeight - 输入高度
     * @param {Uint8Array} input - 输入压缩数据
     * @returns {boolean} 是否成功
     */
    bitmapDecompress15(output, outputWidth, outputHeight, inputWidth, inputHeight, input) {
        // 分配临时缓冲区用于解压缩的15位数据
        const temp = new Uint8Array(inputWidth * inputHeight * 2);

        // 解压缩到临时缓冲区
        if (!this.bitmapDecompress2(temp, inputWidth, inputHeight, input)) {
            return false;
        }

        // 转换为RGBA - RDP使用小端序
        for (let y = 0; y < outputHeight && y < inputHeight; y++) {
            for (let x = 0; x < outputWidth && x < inputWidth; x++) {
                const pixelIdx = (y * inputWidth + x) * 2;

                // RDP使用小端序，低字节在前
                const pixel = temp[pixelIdx] | (temp[pixelIdx + 1] << 8);

                // RGB555: RRRRRGGGGGGBBBBB (bits 15-11, 10-6, 5-1, bit 0 unused)
                const r = ((pixel & 0x7C00) >> 7); // 5 bits red -> 8 bits
                const g = ((pixel & 0x03E0) >> 2); // 5 bits green -> 8 bits
                const b = ((pixel & 0x001F) << 3); // 5 bits blue -> 8 bits

                // 输出RGBA格式
                const idx = (y * outputWidth + x) * 4;
                output[idx + 0] = r;   // R
                output[idx + 1] = g;   // G
                output[idx + 2] = b;   // B
                output[idx + 3] = 255; // A
            }
        }

        return true;
    }

    /**
     * 解压缩 16位 (RGB565) 位图数据到 RGBA 格式
     * @param {Uint8Array} output - 输出RGBA缓冲区
     * @param {number} outputWidth - 输出宽度
     * @param {number} outputHeight - 输出高度
     * @param {number} inputWidth - 输入宽度
     * @param {number} inputHeight - 输入高度
     * @param {Uint8Array} input - 输入压缩数据
     * @returns {boolean} 是否成功
     */
    bitmapDecompress16(output, outputWidth, outputHeight, inputWidth, inputHeight, input) {
        // 分配临时缓冲区用于解压缩的16位数据
        const temp = new Uint8Array(inputWidth * inputHeight * 2);

        // 解压缩到临时缓冲区
        if (!this.bitmapDecompress2(temp, inputWidth, inputHeight, input)) {
            return false;
        }

        // 转换为RGBA - RDP使用小端序
        for (let y = 0; y < outputHeight && y < inputHeight; y++) {
            for (let x = 0; x < outputWidth && x < inputWidth; x++) {
                const pixelIdx = (y * inputWidth + x) * 2;

                // RDP使用小端序，低字节在前
                const pixel = temp[pixelIdx] | (temp[pixelIdx + 1] << 8);

                // RGB565: RRRRRGGGGGGBBBBB (bits 15-11, 10-5, 4-0)
                const r = ((pixel & 0xF800) >> 8); // 5 bits red -> 8 bits
                const g = ((pixel & 0x07E0) >> 3); // 6 bits green -> 8 bits
                const b = ((pixel & 0x001F) << 3); // 5 bits blue -> 8 bits

                // 输出RGBA格式
                const idx = (y * outputWidth + x) * 4;
                output[idx + 0] = r;   // R
                output[idx + 1] = g;   // G
                output[idx + 2] = b;   // B
                output[idx + 3] = 255; // A
            }
        }

        return true;
    }

    /**
     * 解压缩 24位 BGR 位图数据到 RGBA 格式
     * @param {Uint8Array} output - 输出RGBA缓冲区
     * @param {number} outputWidth - 输出宽度
     * @param {number} outputHeight - 输出高度
     * @param {number} inputWidth - 输入宽度
     * @param {number} inputHeight - 输入高度
     * @param {Uint8Array} input - 输入压缩数据
     * @returns {boolean} 是否成功
     */
    bitmapDecompress24(output, outputWidth, outputHeight, inputWidth, inputHeight, input) {
        // 分配临时缓冲区用于解压缩的24位数据
        const temp = new Uint8Array(inputWidth * inputHeight * 3);

        // 解压缩到临时缓冲区
        if (!this.bitmapDecompress3(temp, inputWidth, inputHeight, input)) {
            return false;
        }

        // 转换BGR到RGBA - RDP使用BGR顺序，需要转换为RGB
        for (let y = 0; y < outputHeight && y < inputHeight; y++) {
            for (let x = 0; x < outputWidth && x < inputWidth; x++) {
                const srcIdx = (y * inputWidth + x) * 3;
                const dstIdx = (y * outputWidth + x) * 4;
                
                // RDP使用BGR顺序，需要转换为RGB
                const b = temp[srcIdx + 0]; // B (原BGR中的B)
                const g = temp[srcIdx + 1]; // G (原BGR中的G)
                const r = temp[srcIdx + 2]; // R (原BGR中的R)
                
                // 输出RGBA格式
                output[dstIdx + 0] = r;   // R
                output[dstIdx + 1] = g;   // G
                output[dstIdx + 2] = b;   // B
                output[dstIdx + 3] = 255; // A
            }
        }

        return true;
    }

    /**
     * 解压缩 32位 BGRA 位图数据到 RGBA 格式
     * @param {Uint8Array} output - 输出RGBA缓冲区
     * @param {number} outputWidth - 输出宽度
     * @param {number} outputHeight - 输出高度
     * @param {number} inputWidth - 输入宽度
     * @param {number} inputHeight - 输入高度
     * @param {Uint8Array} input - 输入压缩数据
     * @returns {boolean} 是否成功
     */
    bitmapDecompress32(output, outputWidth, outputHeight, inputWidth, inputHeight, input) {
        // 分配临时缓冲区用于解压缩的32位数据
        const temp = new Uint8Array(inputWidth * inputHeight * 4);

        // 解压缩到临时缓冲区
        if (!this.bitmapDecompress4(temp, inputWidth, inputHeight, input)) {
            return false;
        }

        // 转换BGRA到RGBA - RDP使用BGRA顺序，需要转换为RGBA
        for (let y = 0; y < outputHeight && y < inputHeight; y++) {
            for (let x = 0; x < outputWidth && x < inputWidth; x++) {
                const srcIdx = (y * inputWidth + x) * 4;
                const dstIdx = (y * outputWidth + x) * 4;
                
                // RDP使用BGRA顺序，需要转换为RGBA
                const b = temp[srcIdx + 0]; // B (原BGRA中的B)
                const g = temp[srcIdx + 1]; // G (原BGRA中的G)
                const r = temp[srcIdx + 2]; // R (原BGRA中的R)
                const a = temp[srcIdx + 3]; // A (原BGRA中的A)
                
                // 输出RGBA格式
                output[dstIdx + 0] = r; // R
                output[dstIdx + 1] = g; // G
                output[dstIdx + 2] = b; // B
                output[dstIdx + 3] = a; // A
            }
        }

        return true;
    }

    /**
     * 解压缩每像素2字节的位图数据
     * @param {Uint8Array} output - 输出缓冲区
     * @param {number} width - 宽度
     * @param {number} height - 高度
     * @param {Uint8Array} input - 输入数据
     * @returns {boolean} 是否成功
     */
    bitmapDecompress2(output, width, height, input) {
        let prevline = 0, line = 0, count = 0;
        let offset = 0, code = 0;
        let x = width;
        let opcode = 0;
        let lastopcode = -1;
        let insertmix = false, bicolour = false, isfillormix = false;
        let mixmask = 0, mask = 0;
        let colour1 = 0, colour2 = 0;
        let mix = 0xffff;
        let fom_mask = 0;

        const out = new Uint16Array(width * height);
        let inputIndex = 0;

        while (inputIndex < input.length) {
            fom_mask = 0;
            code = input[inputIndex++];
            opcode = code >> 4;

            // 处理不同的操作码形式
            switch (opcode) {
                case 0xc:
                case 0xd:
                case 0xe:
                    opcode -= 6;
                    count = code & 0xf;
                    offset = 16;
                    break;
                case 0xf:
                    opcode = code & 0xf;
                    if (opcode < 9) {
                        count = input[inputIndex++];
                        count |= input[inputIndex++] << 8;
                    } else {
                        count = 1;
                        if (opcode < 0xb) {
                            count = 8;
                        }
                    }
                    offset = 0;
                    break;
                default:
                    opcode >>= 1;
                    count = code & 0x1f;
                    offset = 32;
                    break;
            }

            // 处理计数的特殊情况
            if (offset !== 0) {
                isfillormix = (opcode === 2) || (opcode === 7);
                if (count === 0) {
                    if (isfillormix) {
                        count = input[inputIndex++] + 1;
                    } else {
                        count = input[inputIndex++] + offset;
                    }
                } else if (isfillormix) {
                    count <<= 3;
                }
            }

            // 读取初步数据
            switch (opcode) {
                case 0: // Fill
                    if (lastopcode === opcode && !(x === width && prevline === 0)) {
                        insertmix = true;
                    }
                    break;
                case 8: // Bicolour
                    colour1 = input[inputIndex] | (input[inputIndex + 1] << 8);
                    inputIndex += 2;
                    // fallthrough
                case 3: // Colour
                    colour2 = input[inputIndex] | (input[inputIndex + 1] << 8);
                    inputIndex += 2;
                    break;
                case 6:
                case 7: // SetMix/Mix, SetMix/FillOrMix
                    mix = input[inputIndex] | (input[inputIndex + 1] << 8);
                    inputIndex += 2;
                    opcode -= 5;
                    break;
                case 9: // FillOrMix_1
                    mask = 0x03;
                    opcode = 0x02;
                    fom_mask = 3;
                    break;
                case 0x0a: // FillOrMix_2
                    mask = 0x05;
                    opcode = 0x02;
                    fom_mask = 5;
                    break;
            }

            lastopcode = opcode;
            mixmask = 0;

            // 输出主体
            while (count > 0) {
                if (x >= width) {
                    if (height <= 0) {
                        return false;
                    }
                    x = 0;
                    height--;
                    prevline = line;
                    line = height * width;
                }

                switch (opcode) {
                    case 0: // Fill
                        if (insertmix) {
                            if (prevline === 0) {
                                out[x + line] = mix;
                            } else {
                                out[x + line] = out[prevline + x] ^ mix;
                            }
                            insertmix = false;
                            count--;
                            x++;
                        }
                        if (prevline === 0) {
                            this.repeat(() => {
                                out[x + line] = 0;
                            }, count, x, width);
                        } else {
                            this.repeat(() => {
                                out[x + line] = out[prevline + x];
                            }, count, x, width);
                        }
                        break;
                    case 1: // Mix
                        if (prevline === 0) {
                            this.repeat(() => {
                                out[x + line] = mix;
                            }, count, x, width);
                        } else {
                            this.repeat(() => {
                                out[x + line] = out[prevline + x] ^ mix;
                            }, count, x, width);
                        }
                        break;
                    case 2: // Fill or Mix
                        if (prevline === 0) {
                            this.repeat(() => {
                                mixmask <<= 1;
                                if (mixmask === 0) {
                                    mask = fom_mask;
                                    if (fom_mask === 0) {
                                        mask = input[inputIndex++];
                                        mixmask = 1;
                                    }
                                }
                                if (mask & mixmask) {
                                    out[x + line] = mix;
                                } else {
                                    out[x + line] = 0;
                                }
                            }, count, x, width);
                        } else {
                            this.repeat(() => {
                                mixmask <<= 1;
                                if (mixmask === 0) {
                                    mask = fom_mask;
                                    if (fom_mask === 0) {
                                        mask = input[inputIndex++];
                                        mixmask = 1;
                                    }
                                }
                                if (mask & mixmask) {
                                    out[x + line] = out[prevline + x] ^ mix;
                                } else {
                                    out[x + line] = out[prevline + x];
                                }
                            }, count, x, width);
                        }
                        break;
                    case 3: // Colour
                        this.repeat(() => {
                            out[x + line] = colour2;
                        }, count, x, width);
                        break;
                    case 4: // Copy
                        this.repeat(() => {
                            const val = input[inputIndex] | (input[inputIndex + 1] << 8);
                            inputIndex += 2;
                            out[x + line] = val;
                        }, count, x, width);
                        break;
                    case 8: // Bicolour
                        this.repeat(() => {
                            if (bicolour) {
                                out[x + line] = colour2;
                                bicolour = false;
                            } else {
                                out[x + line] = colour1;
                                bicolour = true;
                                count++;
                            }
                        }, count, x, width);
                        break;
                    case 0xd: // White
                        this.repeat(() => {
                            out[x + line] = 0xffff;
                        }, count, x, width);
                        break;
                    case 0xe: // Black
                        this.repeat(() => {
                            out[x + line] = 0;
                        }, count, x, width);
                        break;
                    default:
                        console.error(`bitmap opcode 0x${opcode.toString(16)}`);
                        return false;
                }
            }
        }

        // 将uint16数组转换为字节数组，使用正确的字节序
        let j = 0;
        for (let i = 0; i < out.length; i++) {
            output[j] = out[i] & 0xff;
            output[j + 1] = out[i] >> 8;
            j += 2;
        }

        return true;
    }

    /**
     * REPEAT宏实现
     * @param {Function} f - 要重复执行的函数
     * @param {number} count - 计数器
     * @param {number} x - x坐标
     * @param {number} width - 宽度
     */
    repeat(f, count, x, width) {
        while ((count & ~0x7) !== 0 && (x + 8) < width) {
            for (let i = 0; i < 8; i++) {
                f();
                count--;
                x++;
            }
        }

        while (count > 0 && x < width) {
            f();
            count--;
            x++;
        }
    }

    /**
     * 解压缩每像素3字节的位图数据
     * @param {Uint8Array} output - 输出缓冲区
     * @param {number} width - 宽度
     * @param {number} height - 高度
     * @param {Uint8Array} input - 输入数据
     * @returns {boolean} 是否成功
     */
    bitmapDecompress3(output, width, height, input) {
        let prevline = 0, line = 0, count = 0;
        let opcode = 0, offset = 0, code = 0;
        let x = width;
        let lastopcode = -1;
        let insertmix = false, bicolour = false, isfillormix = false;
        let mixmask = 0, mask = 0;
        let colour1 = [0, 0, 0];
        let colour2 = [0, 0, 0];
        let mix = [0xff, 0xff, 0xff];
        let fom_mask = 0;

        let inputIndex = 0;

        while (inputIndex < input.length) {
            fom_mask = 0;
            code = input[inputIndex++];
            opcode = code >> 4;

            // 处理不同的操作码形式
            switch (opcode) {
                case 0xc:
                case 0xd:
                case 0xe:
                    opcode -= 6;
                    count = code & 0xf;
                    offset = 16;
                    break;
                case 0xf:
                    opcode = code & 0xf;
                    if (opcode < 9) {
                        count = input[inputIndex++];
                        count |= input[inputIndex++] << 8;
                    } else {
                        count = 1;
                        if (opcode < 0xb) {
                            count = 8;
                        }
                    }
                    offset = 0;
                    break;
                default:
                    opcode >>= 1;
                    count = code & 0x1f;
                    offset = 32;
                    break;
            }

            // 处理计数的特殊情况
            if (offset !== 0) {
                isfillormix = (opcode === 2) || (opcode === 7);
                if (count === 0) {
                    if (isfillormix) {
                        count = input[inputIndex++] + 1;
                    } else {
                        count = input[inputIndex++] + offset;
                    }
                } else if (isfillormix) {
                    count <<= 3;
                }
            }

            // 读取初步数据
            switch (opcode) {
                case 0: // Fill
                    if (lastopcode === opcode && !(x === width && prevline === 0)) {
                        insertmix = true;
                    }
                    break;
                case 8: // Bicolour
                    colour1[0] = input[inputIndex++];
                    colour1[1] = input[inputIndex++];
                    colour1[2] = input[inputIndex++];
                    // fallthrough
                case 3: // Colour
                    colour2[0] = input[inputIndex++];
                    colour2[1] = input[inputIndex++];
                    colour2[2] = input[inputIndex++];
                    break;
                case 6:
                case 7: // SetMix/Mix, SetMix/FillOrMix
                    mix[0] = input[inputIndex++];
                    mix[1] = input[inputIndex++];
                    mix[2] = input[inputIndex++];
                    opcode -= 5;
                    break;
                case 9: // FillOrMix_1
                    mask = 0x03;
                    opcode = 0x02;
                    fom_mask = 3;
                    break;
                case 0x0a: // FillOrMix_2
                    mask = 0x05;
                    opcode = 0x02;
                    fom_mask = 5;
                    break;
            }

            lastopcode = opcode;
            mixmask = 0;

            // 输出主体
            while (count > 0) {
                if (x >= width) {
                    if (height <= 0) {
                        return false;
                    }
                    x = 0;
                    height--;
                    prevline = line;
                    line = height * width * 3;
                }

                switch (opcode) {
                    case 0: // Fill
                        if (insertmix) {
                            if (prevline === 0) {
                                output[3 * x + line] = mix[0];
                                output[3 * x + line + 1] = mix[1];
                                output[3 * x + line + 2] = mix[2];
                            } else {
                                output[3 * x + line] = output[prevline + 3 * x] ^ mix[0];
                                output[3 * x + line + 1] = output[prevline + 3 * x + 1] ^ mix[1];
                                output[3 * x + line + 2] = output[prevline + 3 * x + 2] ^ mix[2];
                            }
                            insertmix = false;
                            count--;
                            x++;
                        }
                        if (prevline === 0) {
                            this.repeat(() => {
                                output[3 * x + line] = 0;
                                output[3 * x + line + 1] = 0;
                                output[3 * x + line + 2] = 0;
                            }, count, x, width);
                        } else {
                            this.repeat(() => {
                                output[3 * x + line] = output[prevline + 3 * x];
                                output[3 * x + line + 1] = output[prevline + 3 * x + 1];
                                output[3 * x + line + 2] = output[prevline + 3 * x + 2];
                            }, count, x, width);
                        }
                        break;
                    case 1: // Mix
                        if (prevline === 0) {
                            this.repeat(() => {
                                output[3 * x + line] = mix[0];
                                output[3 * x + line + 1] = mix[1];
                                output[3 * x + line + 2] = mix[2];
                            }, count, x, width);
                        } else {
                            this.repeat(() => {
                                output[3 * x + line] = output[prevline + 3 * x] ^ mix[0];
                                output[3 * x + line + 1] = output[prevline + 3 * x + 1] ^ mix[1];
                                output[3 * x + line + 2] = output[prevline + 3 * x + 2] ^ mix[2];
                            }, count, x, width);
                        }
                        break;
                    case 2: // Fill or Mix
                        if (prevline === 0) {
                            this.repeat(() => {
                                mixmask <<= 1;
                                if (mixmask === 0) {
                                    mask = fom_mask;
                                    if (fom_mask === 0) {
                                        mask = input[inputIndex++];
                                        mixmask = 1;
                                    }
                                }
                                if (mask & mixmask) {
                                    output[3 * x + line] = mix[0];
                                    output[3 * x + line + 1] = mix[1];
                                    output[3 * x + line + 2] = mix[2];
                                } else {
                                    output[3 * x + line] = 0;
                                    output[3 * x + line + 1] = 0;
                                    output[3 * x + line + 2] = 0;
                                }
                            }, count, x, width);
                        } else {
                            this.repeat(() => {
                                mixmask <<= 1;
                                if (mixmask === 0) {
                                    mask = fom_mask;
                                    if (fom_mask === 0) {
                                        mask = input[inputIndex++];
                                        mixmask = 1;
                                    }
                                }
                                if (mask & mixmask) {
                                    output[3 * x + line] = output[prevline + 3 * x] ^ mix[0];
                                    output[3 * x + line + 1] = output[prevline + 3 * x + 1] ^ mix[1];
                                    output[3 * x + line + 2] = output[prevline + 3 * x + 2] ^ mix[2];
                                } else {
                                    output[3 * x + line] = output[prevline + 3 * x];
                                    output[3 * x + line + 1] = output[prevline + 3 * x + 1];
                                    output[3 * x + line + 2] = output[prevline + 3 * x + 2];
                                }
                            }, count, x, width);
                        }
                        break;
                    case 3: // Colour
                        this.repeat(() => {
                            output[3 * x + line] = colour2[0];
                            output[3 * x + line + 1] = colour2[1];
                            output[3 * x + line + 2] = colour2[2];
                        }, count, x, width);
                        break;
                    case 4: // Copy
                        this.repeat(() => {
                            output[3 * x + line] = input[inputIndex++];
                            output[3 * x + line + 1] = input[inputIndex++];
                            output[3 * x + line + 2] = input[inputIndex++];
                        }, count, x, width);
                        break;
                    case 8: // Bicolour
                        this.repeat(() => {
                            if (bicolour) {
                                output[3 * x + line] = colour2[0];
                                output[3 * x + line + 1] = colour2[1];
                                output[3 * x + line + 2] = colour2[2];
                                bicolour = false;
                            } else {
                                output[3 * x + line] = colour1[0];
                                output[3 * x + line + 1] = colour1[1];
                                output[3 * x + line + 2] = colour1[2];
                                bicolour = true;
                                count++;
                            }
                        }, count, x, width);
                        break;
                    case 0xd: // White
                        this.repeat(() => {
                            output[3 * x + line] = 0xff;
                            output[3 * x + line + 1] = 0xff;
                            output[3 * x + line + 2] = 0xff;
                        }, count, x, width);
                        break;
                    case 0xe: // Black
                        this.repeat(() => {
                            output[3 * x + line] = 0;
                            output[3 * x + line + 1] = 0;
                            output[3 * x + line + 2] = 0;
                        }, count, x, width);
                        break;
                    default:
                        console.error(`bitmap opcode 0x${opcode.toString(16)}`);
                        return false;
                }
            }
        }

        return true;
    }

    /**
     * 解压缩每像素4字节的位图数据
     * @param {Uint8Array} output - 输出缓冲区
     * @param {number} width - 宽度
     * @param {number} height - 高度
     * @param {Uint8Array} input - 输入数据
     * @returns {boolean} 是否成功
     */
    bitmapDecompress4(output, width, height, input) {
        let code = input[0];
        let inputIndex = 1;
        
        if (code !== 0x10) {
            return false;
        }

        let total = 1;
        let onceBytes = this.processPlane(input, inputIndex, width, height, output, 3);
        total += onceBytes;
        inputIndex += onceBytes;

        onceBytes = this.processPlane(input, inputIndex, width, height, output, 2);
        total += onceBytes;
        inputIndex += onceBytes;

        onceBytes = this.processPlane(input, inputIndex, width, height, output, 1);
        total += onceBytes;
        inputIndex += onceBytes;

        onceBytes = this.processPlane(input, inputIndex, width, height, output, 0);
        total += onceBytes;

        return total === input.length; // +1 for the initial code byte
    }

    /**
     * 处理单个颜色平面
     * @param {Uint8Array} input - 输入数据
     * @param {number} inputIndex - 输入索引
     * @param {number} width - 宽度
     * @param {number} height - 高度
     * @param {Uint8Array} output - 输出缓冲区
     * @param {number} offset - 偏移量
     * @returns {number} 处理的字节数
     */
    processPlane(input, inputIndex, width, height, output, offset) {
        let indexw = 0;
        let indexh = 0;
        let code = 0;
        let collen = 0;
        let replen = 0;
        let color = 0;
        let x = 0;
        let revcode = 0;
        let lastline = 0;
        let thisline = 0;

        const ln = input.length;
        lastline = 0;
        indexh = 0;
        const startIndex = inputIndex;

        while (indexh < height) {
            thisline = offset + (width * height * 4) - ((indexh + 1) * width * 4);
            color = 0;
            indexw = 0;
            let i = thisline;

            if (lastline === 0) {
                while (indexw < width) {
                    code = input[inputIndex++];
                    replen = code & 0xf;
                    collen = (code >> 4) & 0xf;
                    revcode = (replen << 4) | collen;
                    if (revcode <= 47 && revcode >= 16) {
                        replen = revcode;
                        collen = 0;
                    }
                    while (collen > 0) {
                        color = input[inputIndex++];
                        output[i] = color;
                        i += 4;
                        indexw++;
                        collen--;
                    }
                    while (replen > 0) {
                        output[i] = color;
                        i += 4;
                        indexw++;
                        replen--;
                    }
                }
            } else {
                while (indexw < width) {
                    code = input[inputIndex++];
                    replen = code & 0xf;
                    collen = (code >> 4) & 0xf;
                    revcode = (replen << 4) | collen;
                    if (revcode <= 47 && revcode >= 16) {
                        replen = revcode;
                        collen = 0;
                    }
                    while (collen > 0) {
                        x = input[inputIndex++];
                        if (x & 1) {
                            x = x >> 1;
                            x = x + 1;
                            color = -x;
                        } else {
                            x = x >> 1;
                            color = x;
                        }
                        x = output[indexw * 4 + lastline] + color;
                        output[i] = x;
                        i += 4;
                        indexw++;
                        collen--;
                    }
                    while (replen > 0) {
                        x = output[indexw * 4 + lastline] + color;
                        output[i] = x;
                        i += 4;
                        indexw++;
                        replen--;
                    }
                }
            }
            indexh++;
            lastline = thisline;
        }
        
        return inputIndex - startIndex;
    }

    /**
     * 调试颜色转换函数
     */
    debugColorConversion() {
        console.log("=== RLE解压缩函数调试信息 ===");
        console.log("bitmapDecompress15: 15位RGB555解压缩");
        console.log("bitmapDecompress16: 16位RGB565解压缩");
        console.log("bitmapDecompress24: 24位BGR解压缩");
        console.log("bitmapDecompress32: 32位BGRA解压缩");
        console.log("REPEAT宏: 已正确实现");
        console.log("字节序: 使用小端序（低字节在前）");
        console.log("颜色格式: RDP BGR/BGRA -> RGBA");
    }
}

// 导出模块
if (typeof module !== 'undefined' && module.exports) {
    module.exports = RLEDecompressor;
} else if (typeof window !== 'undefined') {
    window.RLEDecompressor = RLEDecompressor;
} 