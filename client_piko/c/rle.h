#ifndef RLE_H
#define RLE_H

#ifdef __cplusplus
extern "C" {
#endif

// 类型定义
typedef unsigned char uint8;
typedef unsigned short uint16;
typedef unsigned int uint32;
typedef int RD_BOOL;

// 函数声明
int bitmap_decompress_15(uint8 * output, int output_width, int output_height, int input_width, int input_height, uint8* input, int size);
int bitmap_decompress_16(uint8 * output, int output_width, int output_height, int input_width, int input_height, uint8* input, int size);
int bitmap_decompress_24(uint8 * output, int output_width, int output_height, int input_width, int input_height, uint8* input, int size);
int bitmap_decompress_32(uint8 * output, int output_width, int output_height, int input_width, int input_height, uint8* input, int size);

#ifdef __cplusplus
}
#endif

#endif // RLE_H 