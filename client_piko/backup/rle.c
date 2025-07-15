/* -*- c-basic-offset: 8 -*-
   rdesktop: A Remote Desktop Protocol client.
   Bitmap decompression routines
   Copyright (C) Matthew Chapman <matthewc.unsw.edu.au> 1999-2008

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

/* three seperate function for speed when decompressing the bitmaps
   when modifing one function make the change in the others
   jay.sorg@gmail.com */

/* indent is confused by this file */
/* *INDENT-OFF* */

#include "rle.h"
#include <stdlib.h>

/* Specific rename for RDPY integration */
#ifndef uint8
#define uint8	unsigned char
#endif
#ifndef uint16
#define uint16	unsigned short
#endif
#ifndef uint32
#define uint32	unsigned int
#endif
#define unimpl(str, code)

#ifndef RD_BOOL
#define	 RD_BOOL	int
#endif
#define False	0
#define True	1
/* end specific rename */

#define CVAL(p)   (*(p++))
//#ifdef NEED_ALIGN
//#ifdef L_ENDIAN
#define CVAL2(p, v) { v = (*(p++)); v |= (*(p++)) << 8; }
//#else
//#define CVAL2(p, v) { v = (*(p++)) << 8; v |= (*(p++)); }
//#endif /* L_ENDIAN */
//#else
//#define CVAL2(p, v) { v = (*((uint16*)p)); p += 2; }
//#endif /* NEED_ALIGN */

#define UNROLL8(exp) { exp exp exp exp exp exp exp exp }

#define REPEAT(statement) \
{ \
	while((count & ~0x7) && ((x+8) < width)) \
		UNROLL8( statement; count--; x++; ); \
	\
	while((count > 0) && (x < width)) \
	{ \
		statement; \
		count--; \
		x++; \
	} \
}

#define MASK_UPDATE() \
{ \
	mixmask <<= 1; \
	if (mixmask == 0) \
	{ \
		mask = fom_mask ? fom_mask : CVAL(input); \
		mixmask = 1; \
	} \
}

/* 1 byte bitmap decompress */
static RD_BOOL
bitmap_decompress1(uint8 * output, int width, int height, uint8 * input, int size)
{
	uint8 *end = input + size;
	uint8 *prevline = NULL, *line = NULL;
	int opcode, count, offset, isfillormix, x = width;
	int lastopcode = -1, insertmix = False, bicolour = False;
	uint8 code;
	uint8 colour1 = 0, colour2 = 0;
	uint8 mixmask, mask = 0;
	uint8 mix = 0xff;
	int fom_mask = 0;

	while (input < end)
	{
		fom_mask = 0;
		code = CVAL(input);
		opcode = code >> 4;
		/* Handle different opcode forms */
		switch (opcode)
		{
			case 0xc:
			case 0xd:
			case 0xe:
				opcode -= 6;
				count = code & 0xf;
				offset = 16;
				break;
			case 0xf:
				opcode = code & 0xf;
				if (opcode < 9)
				{
					count = CVAL(input);
					count |= CVAL(input) << 8;
				}
				else
				{
					count = (opcode < 0xb) ? 8 : 1;
				}
				offset = 0;
				break;
			default:
				opcode >>= 1;
				count = code & 0x1f;
				offset = 32;
				break;
		}
		/* Handle strange cases for counts */
		if (offset != 0)
		{
			isfillormix = ((opcode == 2) || (opcode == 7));
			if (count == 0)
			{
				if (isfillormix)
					count = CVAL(input) + 1;
				else
					count = CVAL(input) + offset;
			}
			else if (isfillormix)
			{
				count <<= 3;
			}
		}
		/* Read preliminary data */
		switch (opcode)
		{
			case 0:	/* Fill */
				if ((lastopcode == opcode) && !((x == width) && (prevline == NULL)))
					insertmix = True;
				break;
			case 8:	/* Bicolour */
				colour1 = CVAL(input);
			case 3:	/* Colour */
				colour2 = CVAL(input);
				break;
			case 6:	/* SetMix/Mix */
			case 7:	/* SetMix/FillOrMix */
				mix = CVAL(input);
				opcode -= 5;
				break;
			case 9:	/* FillOrMix_1 */
				mask = 0x03;
				opcode = 0x02;
				fom_mask = 3;
				break;
			case 0x0a:	/* FillOrMix_2 */
				mask = 0x05;
				opcode = 0x02;
				fom_mask = 5;
				break;
		}
		lastopcode = opcode;
		mixmask = 0;
		/* Output body */
		while (count > 0)
		{
			if (x >= width)
			{
				if (height <= 0)
					return False;
				x = 0;
				height--;
				prevline = line;
				line = output + height * width;
			}
			switch (opcode)
			{
				case 0:	/* Fill */
					if (insertmix)
					{
						if (prevline == NULL)
							line[x] = mix;
						else
							line[x] = prevline[x] ^ mix;
						insertmix = False;
						count--;
						x++;
					}
					if (prevline == NULL)
					{
						REPEAT(line[x] = 0)
					}
					else
					{
						REPEAT(line[x] = prevline[x])
					}
					break;
				case 1:	/* Mix */
					if (prevline == NULL)
					{
						REPEAT(line[x] = mix)
					}
					else
					{
						REPEAT(line[x] = prevline[x] ^ mix)
					}
					break;
				case 2:	/* Fill or Mix */
					if (prevline == NULL)
					{
						REPEAT
						(
							MASK_UPDATE();
							if (mask & mixmask)
								line[x] = mix;
							else
								line[x] = 0;
						)
					}
					else
					{
						REPEAT
						(
							MASK_UPDATE();
							if (mask & mixmask)
								line[x] = prevline[x] ^ mix;
							else
								line[x] = prevline[x];
						)
					}
					break;
				case 3:	/* Colour */
					REPEAT(line[x] = colour2)
					break;
				case 4:	/* Copy */
					REPEAT(line[x] = CVAL(input))
					break;
				case 8:	/* Bicolour */
					REPEAT
					(
						if (bicolour)
						{
							line[x] = colour2;
							bicolour = False;
						}
						else
						{
							line[x] = colour1;
							bicolour = True; count++;
						}
					)
					break;
				case 0xd:	/* White */
					REPEAT(line[x] = 0xff)
					break;
				case 0xe:	/* Black */
					REPEAT(line[x] = 0)
					break;
				default:
					unimpl("bitmap opcode 0x%x\n", opcode);
					return False;
			}
		}
	}
	return True;
}

/* 2 byte bitmap decompress */
static RD_BOOL
bitmap_decompress2(uint8 * output, int width, int height, uint8 * input, int size)
{
	uint8 *end = input + size;
	uint16 *prevline = NULL, *line = NULL;
	int opcode, count, offset, isfillormix, x = width;
	int lastopcode = -1, insertmix = False, bicolour = False;
	uint8 code;
	uint16 colour1 = 0, colour2 = 0;
	uint8 mixmask, mask = 0;
	uint16 mix = 0xffff;
	int fom_mask = 0;

	while (input < end)
	{
		fom_mask = 0;
		code = CVAL(input);
		opcode = code >> 4;
		/* Handle different opcode forms */
		switch (opcode)
		{
			case 0xc:
			case 0xd:
			case 0xe:
				opcode -= 6;
				count = code & 0xf;
				offset = 16;
				break;
			case 0xf:
				opcode = code & 0xf;
				if (opcode < 9)
				{
					count = CVAL(input);
					count |= CVAL(input) << 8;
				}
				else
				{
					count = (opcode < 0xb) ? 8 : 1;
				}
				offset = 0;
				break;
			default:
				opcode >>= 1;
				count = code & 0x1f;
				offset = 32;
				break;
		}
		/* Handle strange cases for counts */
		if (offset != 0)
		{
			isfillormix = ((opcode == 2) || (opcode == 7));
			if (count == 0)
			{
				if (isfillormix)
					count = CVAL(input) + 1;
				else
					count = CVAL(input) + offset;
			}
			else if (isfillormix)
			{
				count <<= 3;
			}
		}
		/* Read preliminary data */
		switch (opcode)
		{
			case 0:	/* Fill */
				if ((lastopcode == opcode) && !((x == width) && (prevline == NULL)))
					insertmix = True;
				break;
			case 8:	/* Bicolour */
				CVAL2(input, colour1);
			case 3:	/* Colour */
				CVAL2(input, colour2);
				break;
			case 6:	/* SetMix/Mix */
			case 7:	/* SetMix/FillOrMix */
				CVAL2(input, mix);
				opcode -= 5;
				break;
			case 9:	/* FillOrMix_1 */
				mask = 0x03;
				opcode = 0x02;
				fom_mask = 3;
				break;
			case 0x0a:	/* FillOrMix_2 */
				mask = 0x05;
				opcode = 0x02;
				fom_mask = 5;
				break;
		}
		lastopcode = opcode;
		mixmask = 0;
		/* Output body */
		while (count > 0)
		{
			if (x >= width)
			{
				if (height <= 0)
					return False;
				x = 0;
				height--;
				prevline = line;
				line = ((uint16 *) output) + height * width;
			}
			switch (opcode)
			{
				case 0:	/* Fill */
					if (insertmix)
					{
						if (prevline == NULL)
							line[x] = mix;
						else
							line[x] = prevline[x] ^ mix;
						insertmix = False;
						count--;
						x++;
					}
					if (prevline == NULL)
					{
						REPEAT(line[x] = 0)
					}
					else
					{
						REPEAT(line[x] = prevline[x])
					}
					break;
				case 1:	/* Mix */
					if (prevline == NULL)
					{
						REPEAT(line[x] = mix)
					}
					else
					{
						REPEAT(line[x] = prevline[x] ^ mix)
					}
					break;
				case 2:	/* Fill or Mix */
					if (prevline == NULL)
					{
						REPEAT
						(
							MASK_UPDATE();
							if (mask & mixmask)
								line[x] = mix;
							else
								line[x] = 0;
						)
					}
					else
					{
						REPEAT
						(
							MASK_UPDATE();
							if (mask & mixmask)
								line[x] = prevline[x] ^ mix;
							else
								line[x] = prevline[x];
						)
					}
					break;
				case 3:	/* Colour */
					REPEAT(line[x] = colour2)
					break;
				case 4:	/* Copy */
					REPEAT(CVAL2(input, line[x]))
					break;
				case 8:	/* Bicolour */
					REPEAT
					(
						if (bicolour)
						{
							line[x] = colour2;
							bicolour = False;
						}
						else
						{
							line[x] = colour1;
							bicolour = True;
							count++;
						}
					)
					break;
				case 0xd:	/* White */
					REPEAT(line[x] = 0xffff)
					break;
				case 0xe:	/* Black */
					REPEAT(line[x] = 0)
					break;
				default:
					unimpl("bitmap opcode 0x%x\n", opcode);
					return False;
			}
		}
	}
	return True;
}

/* 3 byte bitmap decompress */
static RD_BOOL
bitmap_decompress3(uint8 * output, int width, int height, uint8 * input, int size)
{
	uint8 *end = input + size;
	uint8 *prevline = NULL, *line = NULL;
	int opcode, count, offset, isfillormix, x = width;
	int lastopcode = -1, insertmix = False, bicolour = False;
	uint8 code;
	uint8 colour1[3] = {0, 0, 0}, colour2[3] = {0, 0, 0};
	uint8 mixmask, mask = 0;
	uint8 mix[3] = {0xff, 0xff, 0xff};
	int fom_mask = 0;

	while (input < end)
	{
		fom_mask = 0;
		code = CVAL(input);
		opcode = code >> 4;
		/* Handle different opcode forms */
		switch (opcode)
		{
			case 0xc:
			case 0xd:
			case 0xe:
				opcode -= 6;
				count = code & 0xf;
				offset = 16;
				break;
			case 0xf:
				opcode = code & 0xf;
				if (opcode < 9)
				{
					count = CVAL(input);
					count |= CVAL(input) << 8;
				}
				else
				{
					count = (opcode <
						 0xb) ? 8 : 1;
				}
				offset = 0;
				break;
			default:
				opcode >>= 1;
				count = code & 0x1f;
				offset = 32;
				break;
		}
		/* Handle strange cases for counts */
		if (offset != 0)
		{
			isfillormix = ((opcode == 2) || (opcode == 7));
			if (count == 0)
			{
				if (isfillormix)
					count = CVAL(input) + 1;
				else
					count = CVAL(input) + offset;
			}
			else if (isfillormix)
			{
				count <<= 3;
			}
		}
		/* Read preliminary data */
		switch (opcode)
		{
			case 0:	/* Fill */
				if ((lastopcode == opcode) && !((x == width) && (prevline == NULL)))
					insertmix = True;
				break;
			case 8:	/* Bicolour */
				colour1[0] = CVAL(input);
				colour1[1] = CVAL(input);
				colour1[2] = CVAL(input);
			case 3:	/* Colour */
				colour2[0] = CVAL(input);
				colour2[1] = CVAL(input);
				colour2[2] = CVAL(input);
				break;
			case 6:	/* SetMix/Mix */
			case 7:	/* SetMix/FillOrMix */
				mix[0] = CVAL(input);
				mix[1] = CVAL(input);
				mix[2] = CVAL(input);
				opcode -= 5;
				break;
			case 9:	/* FillOrMix_1 */
				mask = 0x03;
				opcode = 0x02;
				fom_mask = 3;
				break;
			case 0x0a:	/* FillOrMix_2 */
				mask = 0x05;
				opcode = 0x02;
				fom_mask = 5;
				break;
		}
		lastopcode = opcode;
		mixmask = 0;
		/* Output body */
		while (count > 0)
		{
			if (x >= width)
			{
				if (height <= 0)
					return False;
				x = 0;
				height--;
				prevline = line;
				line = output + height * (width * 3);
			}
			switch (opcode)
			{
				case 0:	/* Fill */
					if (insertmix)
					{
						if (prevline == NULL)
						{
							line[x * 3] = mix[0];
							line[x * 3 + 1] = mix[1];
							line[x * 3 + 2] = mix[2];
						}
						else
						{
							line[x * 3] =
							 prevline[x * 3] ^ mix[0];
							line[x * 3 + 1] =
							 prevline[x * 3 + 1] ^ mix[1];
							line[x * 3 + 2] =
							 prevline[x * 3 + 2] ^ mix[2];
						}
						insertmix = False;
						count--;
						x++;
					}
					if (prevline == NULL)
					{
						REPEAT
						(
							line[x * 3] = 0;
							line[x * 3 + 1] = 0;
							line[x * 3 + 2] = 0;
						)
					}
					else
					{
						REPEAT
						(
							line[x * 3] = prevline[x * 3];
							line[x * 3 + 1] = prevline[x * 3 + 1];
							line[x * 3 + 2] = prevline[x * 3 + 2];
						)
					}
					break;
				case 1:	/* Mix */
					if (prevline == NULL)
					{
						REPEAT
						(
							line[x * 3] = mix[0];
							line[x * 3 + 1] = mix[1];
							line[x * 3 + 2] = mix[2];
						)
					}
					else
					{
						REPEAT
						(
							line[x * 3] =
							 prevline[x * 3] ^ mix[0];
							line[x * 3 + 1] =
							 prevline[x * 3 + 1] ^ mix[1];
							line[x * 3 + 2] =
							 prevline[x * 3 + 2] ^ mix[2];
						)
					}
					break;
				case 2:	/* Fill or Mix */
					if (prevline == NULL)
					{
						REPEAT
						(
							MASK_UPDATE();
							if (mask & mixmask)
							{
								line[x * 3] = mix[0];
								line[x * 3 + 1] = mix[1];
								line[x * 3 + 2] = mix[2];
							}
							else
							{
								line[x * 3] = 0;
								line[x * 3 + 1] = 0;
								line[x * 3 + 2] = 0;
							}
						)
					}
					else
					{
						REPEAT
						(
							MASK_UPDATE();
							if (mask & mixmask)
							{
								line[x * 3] =
								 prevline[x * 3] ^ mix [0];
								line[x * 3 + 1] =
								 prevline[x * 3 + 1] ^ mix [1];
								line[x * 3 + 2] =
								 prevline[x * 3 + 2] ^ mix [2];
							}
							else
							{
								line[x * 3] =
								 prevline[x * 3];
								line[x * 3 + 1] =
								 prevline[x * 3 + 1];
								line[x * 3 + 2] =
								 prevline[x * 3 + 2];
							}
						)
					}
					break;
				case 3:	/* Colour */
					REPEAT
					(
						line[x * 3] = colour2 [0];
						line[x * 3 + 1] = colour2 [1];
						line[x * 3 + 2] = colour2 [2];
					)
					break;
				case 4:	/* Copy */
					REPEAT
					(
						line[x * 3] = CVAL(input);
						line[x * 3 + 1] = CVAL(input);
						line[x * 3 + 2] = CVAL(input);
					)
					break;
				case 8:	/* Bicolour */
					REPEAT
					(
						if (bicolour)
						{
							line[x * 3] = colour2[0];
							line[x * 3 + 1] = colour2[1];
							line[x * 3 + 2] = colour2[2];
							bicolour = False;
						}
						else
						{
							line[x * 3] = colour1[0];
							line[x * 3 + 1] = colour1[1];
							line[x * 3 + 2] = colour1[2];
							bicolour = True;
							count++;
						}
					)
					break;
				case 0xd:	/* White */
					REPEAT
					(
						line[x * 3] = 0xff;
						line[x * 3 + 1] = 0xff;
						line[x * 3 + 2] = 0xff;
					)
					break;
				case 0xe:	/* Black */
					REPEAT
					(
						line[x * 3] = 0;
						line[x * 3 + 1] = 0;
						line[x * 3 + 2] = 0;
					)
					break;
				default:
					unimpl("bitmap opcode 0x%x\n", opcode);
					return False;
			}
		}
	}
	return True;
}

/* decompress a colour plane */
static int
process_plane(uint8 * in, int width, int height, uint8 * out, int size)
{
	int indexw;
	int indexh;
	int code;
	int collen;
	int replen;
	int color;
	int x;
	int revcode;
	uint8 * last_line;
	uint8 * this_line;
	uint8 * org_in;
	uint8 * org_out;

	org_in = in;
	org_out = out;
	last_line = 0;
	indexh = 0;
	while (indexh < height)
	{
		out = (org_out + width * height * 4) - ((indexh + 1) * width * 4);
		color = 0;
		this_line = out;
		indexw = 0;
		if (last_line == 0)
		{
			while (indexw < width)
			{
				code = CVAL(in);
				replen = code & 0xf;
				collen = (code >> 4) & 0xf;
				revcode = (replen << 4) | collen;
				if ((revcode <= 47) && (revcode >= 16))
				{
					replen = revcode;
					collen = 0;
				}
				while (collen > 0)
				{
					color = CVAL(in);
					*out = color;
					out += 4;
					indexw++;
					collen--;
				}
				while (replen > 0)
				{
					*out = color;
					out += 4;
					indexw++;
					replen--;
				}
			}
		}
		else
		{
			while (indexw < width)
			{
				code = CVAL(in);
				replen = code & 0xf;
				collen = (code >> 4) & 0xf;
				revcode = (replen << 4) | collen;
				if ((revcode <= 47) && (revcode >= 16))
				{
					replen = revcode;
					collen = 0;
				}
				while (collen > 0)
				{
					x = CVAL(in);
					if (x & 1)
					{
						x = x >> 1;
						x = x + 1;
						color = -x;
					}
					else
					{
						x = x >> 1;
						color = x;
					}
					x = last_line[indexw * 4] + color;
					*out = x;
					out += 4;
					indexw++;
					collen--;
				}
				while (replen > 0)
				{
					x = last_line[indexw * 4] + color;
					*out = x;
					out += 4;
					indexw++;
					replen--;
				}
			}
		}
		indexh++;
		last_line = this_line;
	}
	return (int) (in - org_in);
}

/* 4 byte bitmap decompress */
static RD_BOOL
bitmap_decompress4(uint8 * output, int width, int height, uint8 * input, int size)
{
	int code;
	int bytes_pro;
	int total_pro;

	code = CVAL(input);
	if (code != 0x10)
	{
		return False;
	}
	total_pro = 1;
	bytes_pro = process_plane(input, width, height, output + 3, size - total_pro);
	total_pro += bytes_pro;
	input += bytes_pro;
	bytes_pro = process_plane(input, width, height, output + 2, size - total_pro);
	total_pro += bytes_pro;
	input += bytes_pro;
	bytes_pro = process_plane(input, width, height, output + 1, size - total_pro);
	total_pro += bytes_pro;
	input += bytes_pro;
	bytes_pro = process_plane(input, width, height, output + 0, size - total_pro);
	total_pro += bytes_pro;
	return size == total_pro;
}

int
bitmap_decompress_15(uint8 * output, int output_width, int output_height, int input_width, int input_height, uint8* input, int size) {
	// 分配临时缓冲区用于存储解压缩的15位数据
	uint8 * temp = (uint8*)malloc(input_width * input_height * 2);
	if (!temp) {
		return 0; // 内存分配失败
	}
	
	RD_BOOL rv = bitmap_decompress2(temp, input_width, input_height, input, size);
	if (!rv) {
		free(temp);
		return 0;
	}

	// convert to rgba - 修复：使用正确的字节序和颜色格式
	for (int y = 0; y < output_height && y < input_height; y++) {
		for (int x = 0; x < output_width && x < input_width; x++) {
			// 修复：RDP使用小端序，所以低字节在前
			uint16 pixel = ((uint16*)temp)[y * input_width + x];
			// 修复：正确的15位颜色格式 (RGB555)
			uint8 r = (pixel & 0x7c00) >> 10;  // 5位红色
			uint8 g = (pixel & 0x03e0) >> 5;   // 5位绿色
			uint8 b = (pixel & 0x001f);        // 5位蓝色
			// 转换为8位
			r = r * 255 / 31;
			g = g * 255 / 31;
			b = b * 255 / 31;
			// 修复：输出RGBA格式，而不是BGRA
			output[(y * output_width + x) * 4 + 0] = r;  // R
			output[(y * output_width + x) * 4 + 1] = g;  // G
			output[(y * output_width + x) * 4 + 2] = b;  // B
			output[(y * output_width + x) * 4 + 3] = 255; // A
		}
	}

	free(temp);
	return rv;
}

int
bitmap_decompress_16(uint8 * output, int output_width, int output_height, int input_width, int input_height, uint8* input, int size) {
	// 分配临时缓冲区用于存储解压缩的16位数据
	uint8 * temp = (uint8*)malloc(input_width * input_height * 2);
	if (!temp) {
		return 0; // 内存分配失败
	}
	
	RD_BOOL rv = bitmap_decompress2(temp, input_width, input_height, input, size);
	if (!rv) {
		free(temp);
		return 0;
	}

	// convert to rgba - 修复：使用正确的字节序和颜色格式
	for (int y = 0; y < output_height && y < input_height; y++) {
		for (int x = 0; x < output_width && x < input_width; x++) {
			// 修复：RDP使用小端序，所以低字节在前
			uint16 pixel = ((uint16*)temp)[y * input_width + x];
			// 修复：正确的16位颜色格式 (RGB565)
			uint8 r = (pixel & 0xf800) >> 11;  // 5位红色
			uint8 g = (pixel & 0x07e0) >> 5;   // 6位绿色
			uint8 b = (pixel & 0x001f);        // 5位蓝色
			// 转换为8位
			r = r * 255 / 31;
			g = g * 255 / 63;
			b = b * 255 / 31;
			// 修复：输出RGBA格式，而不是BGRA
			output[(y * output_width + x) * 4 + 0] = r;  // R
			output[(y * output_width + x) * 4 + 1] = g;  // G
			output[(y * output_width + x) * 4 + 2] = b;  // B
			output[(y * output_width + x) * 4 + 3] = 255; // A
		}
	}
	free(temp);
	return rv;
}

int
bitmap_decompress_24(uint8 * output, int output_width, int output_height, int input_width, int input_height, uint8* input, int size) {
	// 分配临时缓冲区用于存储解压缩的24位数据
	uint8 * temp = (uint8*)malloc(input_width * input_height * 3);
	if (!temp) {
		return 0; // 内存分配失败
	}
	
	RD_BOOL rv = bitmap_decompress3(temp, input_width, input_height, input, size);
	if (!rv) {
		free(temp);
		return 0;
	}

	// convert to rgba - 修复：RDP使用BGR顺序，需要转换为RGB
	for (int y = 0; y < output_height && y < input_height; y++) {
		for (int x = 0; x < output_width && x < input_width; x++) {
			// 修复：RDP使用BGR顺序，需要转换为RGB
			uint8 b = temp[(y * input_width + x) * 3 + 0];  // B (原BGR中的B)
			uint8 g = temp[(y * input_width + x) * 3 + 1];  // G (原BGR中的G)
			uint8 r = temp[(y * input_width + x) * 3 + 2];  // R (原BGR中的R)
			// 修复：输出RGBA格式
			output[(y * output_width + x) * 4 + 0] = r;  // R
			output[(y * output_width + x) * 4 + 1] = g;  // G
			output[(y * output_width + x) * 4 + 2] = b;  // B
			output[(y * output_width + x) * 4 + 3] = 255; // A
		}
	}
	free(temp);

	return rv;
}

int
bitmap_decompress_32(uint8 * output, int output_width, int output_height, int input_width, int input_height, uint8* input, int size) {
	// 分配临时缓冲区用于存储解压缩的32位数据
	uint8 * temp = (uint8*)malloc(input_width * input_height * 4);
	if (!temp) {
		return 0; // 内存分配失败
	}
	
	RD_BOOL rv = bitmap_decompress4(temp, input_width, input_height, input, size);
	if (!rv) {
		free(temp);
		return 0;
	}

	// convert to rgba - 修复：RDP使用BGRA顺序，需要转换为RGBA
	for (int y = 0; y < output_height && y < input_height; y++) {
		for (int x = 0; x < output_width && x < input_width; x++) {
			// 修复：RDP使用BGRA顺序，需要转换为RGBA
			uint8 b = temp[(y * input_width + x) * 4 + 0];  // B (原BGRA中的B)
			uint8 g = temp[(y * input_width + x) * 4 + 1];  // G (原BGRA中的G)
			uint8 r = temp[(y * input_width + x) * 4 + 2];  // R (原BGRA中的R)
			uint8 a = temp[(y * input_width + x) * 4 + 3];  // A (原BGRA中的A)
			// 修复：输出RGBA格式
			output[(y * output_width + x) * 4 + 0] = r;  // R
			output[(y * output_width + x) * 4 + 1] = g;  // G
			output[(y * output_width + x) * 4 + 2] = b;  // B
			output[(y * output_width + x) * 4 + 3] = a;  // A
		}
	}
	free(temp);

	return rv;
}