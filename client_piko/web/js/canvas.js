/*
 * Copyright (c) 2015 Sylvain Peyrefitte
 *
 * This file is part of mstsc.js.
 *
 * mstsc.js is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

(function() {
	
	/**
	 * decompress bitmap from RLE algorithm
	 * @param	bitmap	{object} bitmap object of bitmap event of node-rdpjs
	 */
	function decompress (bitmap) {
		console.log('=== [canvas.js] 开始处理bitmap ===');
		console.log('[canvas.js] bitmap详情:', {
			width: bitmap.width,
			height: bitmap.height,
			bitsPerPixel: bitmap.bitsPerPixel,
			isCompress: bitmap.isCompress,
			dataLength: bitmap.data ? bitmap.data.length : 0,
			destLeft: bitmap.destLeft,
			destTop: bitmap.destTop,
			destRight: bitmap.destRight,
			destBottom: bitmap.destBottom
		});

		// 检查输入数据
		if (!bitmap.data || bitmap.data.length === 0) {
			console.error('[canvas.js] No bitmap data provided');
			showDecompressError('没有提供位图数据');
			return null;
		}

		// 验证基本参数
		if (bitmap.width <= 0 || bitmap.height <= 0) {
			console.error('[canvas.js] Invalid bitmap dimensions:', bitmap.width, 'x', bitmap.height);
			showDecompressError('位图尺寸无效');
			return null;
		}

		// 验证位深度
		if (bitmap.bitsPerPixel !== 15 && bitmap.bitsPerPixel !== 16 && bitmap.bitsPerPixel !== 24 && bitmap.bitsPerPixel !== 32) {
			console.error('[canvas.js] Unsupported bits per pixel:', bitmap.bitsPerPixel);
			showDecompressError('不支持的位深度: ' + bitmap.bitsPerPixel + '位');
			return null;
		}

		// 如果标记为压缩数据，进行RLE解压缩
		if (bitmap.isCompress) {
			console.log('[canvas.js] 检测到压缩数据，开始RLE解压缩...');
			return decompressRLE(bitmap);
		} else {
			// 后端已经解压缩，前端直接处理未压缩数据
			console.log('[canvas.js] 后端已解压缩，直接处理未压缩数据');
			return processUncompressedData(bitmap);
		}
	}

	// RLE解压缩函数
	function decompressRLE(bitmap) {
		console.log('[canvas.js] 开始RLE解压缩，位深度:', bitmap.bitsPerPixel);
		
		try {
			// 计算输出尺寸
			var output_width = bitmap.destRight - bitmap.destLeft + 1;
			var output_height = bitmap.destBottom - bitmap.destTop + 1;
			
			// 验证输出尺寸的合理性
			if (output_width <= 0 || output_height <= 0) {
				console.error('[canvas.js] Invalid output dimensions for RLE decompression:', output_width, 'x', output_height);
				return null;
			}
			
			// 根据位深度选择解压缩方法
			var decompressedData = null;
			switch (bitmap.bitsPerPixel) {
				case 15:
					decompressedData = decompressRLE15(bitmap.data, bitmap.width, bitmap.height, output_width, output_height);
					break;
				case 16:
					decompressedData = decompressRLE16(bitmap.data, bitmap.width, bitmap.height, output_width, output_height);
					break;
				case 24:
					decompressedData = decompressRLE24(bitmap.data, bitmap.width, bitmap.height, output_width, output_height);
					break;
				case 32:
					decompressedData = decompressRLE32(bitmap.data, bitmap.width, bitmap.height, output_width, output_height);
					break;
				default:
					console.error('[canvas.js] Unsupported bits per pixel for RLE decompression:', bitmap.bitsPerPixel);
					return null;
			}
			
			if (!decompressedData) {
				console.error('[canvas.js] RLE解压缩失败');
				// 显示提示信息，建议使用后端解压缩
				showDecompressError('前端RLE解压缩暂未实现，建议设置后端解压缩');
				return null;
			}
			
			console.log('[canvas.js] RLE解压缩成功，输出数据:', {
				width: output_width,
				height: output_height,
				dataLength: decompressedData.length,
				expectedSize: output_width * output_height * 4
			});
			
			return { width: output_width, height: output_height, data: decompressedData };
		} catch (e) {
			console.error('[canvas.js] RLE解压缩异常:', e);
			showDecompressError('RLE解压缩异常: ' + e.message);
			return null;
		}
	}

	// 15位RLE解压缩
	function decompressRLE15(input, inputWidth, inputHeight, outputWidth, outputHeight) {
		console.log('[canvas.js] 15位RLE解压缩 - 暂未实现');
		console.warn('[canvas.js] 建议设置 DecompressOnBackend=true 使用后端解压缩');
		return null;
	}

	// 16位RLE解压缩
	function decompressRLE16(input, inputWidth, inputHeight, outputWidth, outputHeight) {
		console.log('[canvas.js] 16位RLE解压缩 - 暂未实现');
		console.warn('[canvas.js] 建议设置 DecompressOnBackend=true 使用后端解压缩');
		return null;
	}

	// 24位RLE解压缩
	function decompressRLE24(input, inputWidth, inputHeight, outputWidth, outputHeight) {
		console.log('[canvas.js] 24位RLE解压缩 - 暂未实现');
		console.warn('[canvas.js] 建议设置 DecompressOnBackend=true 使用后端解压缩');
		return null;
	}

	// 32位RLE解压缩
	function decompressRLE32(input, inputWidth, inputHeight, outputWidth, outputHeight) {
		console.log('[canvas.js] 32位RLE解压缩 - 暂未实现');
		console.warn('[canvas.js] 建议设置 DecompressOnBackend=true 使用后端解压缩');
		return null;
	}

	// 处理未压缩数据的函数
	function processUncompressedData(bitmap) {
		console.log('[canvas.js] 处理后端解压缩的RGBA数据...');
		
		try {
			// 修复：使用正确的输出尺寸计算
			var output_width = bitmap.destRight - bitmap.destLeft + 1;
			var output_height = bitmap.destBottom - bitmap.destTop + 1;
			
			// 验证输出尺寸的合理性
			if (output_width <= 0 || output_height <= 0) {
				console.error('[canvas.js] Invalid output dimensions for uncompressed data:', output_width, 'x', output_height);
				console.error('[canvas.js] destLeft:', bitmap.destLeft, 'destTop:', bitmap.destTop, 'destRight:', bitmap.destRight, 'destBottom:', bitmap.destBottom);
				return null;
			}
			
			var expectedDataLength = output_width * output_height * 4; // RGBA格式
			
			// 验证输入数据长度
			console.log('[canvas.js] RGBA数据长度验证:', {
				actualLength: bitmap.data.length,
				expectedLength: expectedDataLength,
				outputWidth: output_width,
				outputHeight: output_height,
				bytesPerPixel: 4
			});
			
			// 如果数据长度不匹配，尝试调整
			var input = new Uint8Array(bitmap.data);
			if (input.length !== expectedDataLength) {
				console.warn('[canvas.js] 输入数据长度不匹配，尝试调整');
				console.warn('[canvas.js] 期望长度:', expectedDataLength, '实际长度:', input.length);
				
				if (input.length > expectedDataLength) {
					console.warn('[canvas.js] 数据太长，截断到期望长度');
					input = input.slice(0, expectedDataLength);
				} else if (input.length < expectedDataLength) {
					console.warn('[canvas.js] 数据太短，填充到期望长度');
					var paddedInput = new Uint8Array(expectedDataLength);
					paddedInput.set(input);
					// 用0填充剩余部分
					for (var i = input.length; i < expectedDataLength; i++) {
						paddedInput[i] = 0;
					}
					input = paddedInput;
				}
			}
			
			// 后端已经输出RGBA格式，直接使用
			var output = new Uint8ClampedArray(input);
			
			console.log('[canvas.js] RGBA数据处理成功，输出数据:', {
				width: output_width,
				height: output_height,
				dataLength: output.length,
				expectedSize: expectedDataLength,
				processedPixels: Math.floor(output.length / 4)
			});
			
			return { width: output_width, height: output_height, data: output };
		} catch (e) {
			console.error('[canvas.js] 处理RGBA数据失败:', e);
			return null;
		}
	}
	
	// 显示解压缩错误信息的函数
	function showDecompressError(message) {
		// 检查是否在浏览器环境中
		if (typeof document !== 'undefined' && typeof window !== 'undefined') {
			// 创建一个错误提示框
			var errorDiv = document.createElement('div');
			errorDiv.style.cssText = 'position: fixed; top: 20px; right: 20px; background: #fff3cd; color: #856404; padding: 15px; border-radius: 5px; border: 1px solid #ffeaa7; z-index: 1000; max-width: 400px; font-size: 14px; box-shadow: 0 2px 10px rgba(0,0,0,0.1);';
			errorDiv.innerHTML = '<strong>解压缩提示:</strong><br>' + message + '<br><br><strong>解决方案:</strong><br>设置 <code>DecompressOnBackend=true</code> 使用后端解压缩';
			document.body.appendChild(errorDiv);
			
			// 10秒后自动移除提示
			setTimeout(function() {
				if (errorDiv.parentNode) {
					errorDiv.parentNode.removeChild(errorDiv);
				}
			}, 10000);
		}
	}
	
	/**
	 * Un compress bitmap are reverse in y axis
	 */
	function reverse (bitmap) {
		console.log('[canvas.js] Processing RGBA bitmap data:', {
			width: bitmap.width,
			height: bitmap.height,
			dataLength: bitmap.data ? bitmap.data.length : 0,
			bitsPerPixel: bitmap.bitsPerPixel,
			destLeft: bitmap.destLeft,
			destTop: bitmap.destTop,
			destRight: bitmap.destRight,
			destBottom: bitmap.destBottom,
			dataType: bitmap.data ? bitmap.data.constructor.name : 'undefined',
			isArray: Array.isArray(bitmap.data),
			isUint8Array: bitmap.data instanceof Uint8Array,
			isArrayBuffer: bitmap.data instanceof ArrayBuffer
		});
		
		// 添加数据样本检查
		if (bitmap.data && bitmap.data.length > 0) {
			var sampleData = [];
			var sampleCount = Math.min(10, bitmap.data.length);
			for (var i = 0; i < sampleCount; i++) {
				sampleData.push(bitmap.data[i]);
			}
			console.log('[canvas.js] reverse函数输入数据样本:', sampleData);
		} else {
			console.error('[canvas.js] reverse函数输入数据为空或长度为0');
			console.error('[canvas.js] 数据详情:', {
				hasData: !!bitmap.data,
				dataType: typeof bitmap.data,
				constructor: bitmap.data ? bitmap.data.constructor.name : 'undefined',
				length: bitmap.data ? bitmap.data.length : 'undefined',
				isArray: Array.isArray(bitmap.data),
				isUint8Array: bitmap.data instanceof Uint8Array,
				isArrayBuffer: bitmap.data instanceof ArrayBuffer
			});
			return null;
		}
		
		if (!bitmap.data || bitmap.data.length === 0) {
			console.error('[canvas.js] No data in bitmap');
			return null;
		}
		
		// 修复：使用正确的目标尺寸计算
		var target_width = bitmap.destRight - bitmap.destLeft + 1;
		var target_height = bitmap.destBottom - bitmap.destTop + 1;
		
		// 验证目标尺寸的合理性
		if (target_width <= 0 || target_height <= 0) {
			console.error('[canvas.js] Invalid target dimensions:', target_width, 'x', target_height);
			console.error('[canvas.js] destLeft:', bitmap.destLeft, 'destTop:', bitmap.destTop, 'destRight:', bitmap.destRight, 'destBottom:', bitmap.destBottom);
			return null;
		}
		
		// 后端已经统一输出RGBA格式，所以期望长度是4字节/像素
		var expectedLength = target_width * target_height * 4; // RGBA格式
		
		// 添加更详细的调试信息
		console.log('[canvas.js] Length calculation (RGBA format):', {
			targetWidth: target_width,
			targetHeight: target_height,
			bytesPerPixel: 4, // RGBA
			expectedLength: expectedLength,
			actualLength: bitmap.data.length,
			difference: bitmap.data.length - expectedLength
		});
		
		// 如果数据长度不匹配，尝试调整
		var input = new Uint8Array(bitmap.data);
		if (input.length !== expectedLength) {
			console.warn('[canvas.js] 数据长度不匹配，尝试调整');
			console.warn('[canvas.js] 期望长度:', expectedLength, '实际长度:', input.length);
			
			if (input.length > expectedLength) {
				console.warn('[canvas.js] 数据太长，截断到期望长度');
				input = input.slice(0, expectedLength);
			} else if (input.length < expectedLength) {
				console.warn('[canvas.js] 数据太短，填充到期望长度');
				var paddedInput = new Uint8Array(expectedLength);
				paddedInput.set(input);
				// 用0填充剩余部分
				for (var i = input.length; i < expectedLength; i++) {
					paddedInput[i] = 0;
				}
				input = paddedInput;
			}
		}
		
		// 后端已经输出RGBA格式，直接使用
		var output = new Uint8ClampedArray(input);
		
		console.log('[canvas.js] RGBA数据处理成功:', {
			width: target_width,
			height: target_height,
			dataLength: output.length,
			expectedSize: expectedLength,
			processedPixels: Math.floor(output.length / 4)
		});
		
		return { width: target_width, height: target_height, data: output };
	}

	/**
	 * Canvas renderer
	 * @param canvas {canvas} use for rendering
	 */
	function Canvas(canvas) {
		this.canvas = canvas;
		this.ctx = canvas.getContext("2d");
		this.lastUpdateTime = 0; // 添加lastUpdateTime属性
		console.log('[canvas.js] Canvas renderer created');
	}
	
	Canvas.prototype = {
		/**
		 * 检查是否需要触发flush
		 * @returns {boolean}
		 */
		shouldTriggerFlush : function () {
			// 检查canvas尺寸是否发生变化
			var currentWidth = this.canvas.width;
			var currentHeight = this.canvas.height;
			
			// 如果canvas尺寸为0或无效，需要触发flush
			if (currentWidth <= 0 || currentHeight <= 0) {
				console.log('[canvas.js] Canvas尺寸无效，需要触发flush');
				return true;
			}
			
			// 检查canvas是否可见
			if (this.canvas.style.display === 'none') {
				console.log('[canvas.js] Canvas不可见，需要触发flush');
				return true;
			}
			
			// 检查是否长时间没有收到位图更新
			var now = Math.floor(Date.now() / 1000); // 修改为秒级时间戳，与后端对齐
			if (!this.lastUpdateTime) {
				this.lastUpdateTime = now;
				return false;
			}
			
			// 如果超过5秒没有更新，触发flush
			if (now - this.lastUpdateTime > 5) { // 修改为秒级比较
				console.log('[canvas.js] 长时间没有位图更新，需要触发flush');
				return true;
			}
			
			return false;
		},
		
		/**
		 * update canvas with new bitmap
		 * @param bitmap {object}
		 */
		update : function (bitmap) {
			console.log('=== [canvas.js] 开始处理bitmap更新 ===');
			
			// 检查canvas状态
			if (!this.canvas) {
				console.error('[canvas.js] Canvas元素不存在');
				return;
			}
			
			// 检查canvas是否可见
			if (this.canvas.style.display === 'none') {
				console.warn('[canvas.js] Canvas不可见，跳过渲染');
				return;
			}
			
			// 检查canvas尺寸
			if (this.canvas.width <= 0 || this.canvas.height <= 0) {
				console.error('[canvas.js] Canvas尺寸无效:', this.canvas.width, 'x', this.canvas.height);
				return;
			}
			
			// 调用全局调试函数（如果存在）
			if (typeof window.debugCanvasState === 'function') {
				window.debugCanvasState();
			}
			
			// 检查是否需要触发flush
			if (typeof window.triggerRDPFlush === 'function' && this.shouldTriggerFlush()) {
				console.log('[canvas.js] 检测到需要触发flush，执行刷新操作');
				window.triggerRDPFlush();
			}
			
			console.log('[canvas.js] 接收到的bitmap数据:', {
				isCompress: bitmap.isCompress,
				bitsPerPixel: bitmap.bitsPerPixel,
				width: bitmap.width,
				height: bitmap.height,
				destLeft: bitmap.destLeft,
				destTop: bitmap.destTop,
				destRight: bitmap.destRight,
				destBottom: bitmap.destBottom,
				dataLength: bitmap.data ? bitmap.data.length : 0,
				dataType: bitmap.data ? bitmap.data.constructor.name : 'undefined'
			});
			
			// 检查canvas状态
			console.log('[canvas.js] Canvas状态:', {
				canvasWidth: this.canvas.width,
				canvasHeight: this.canvas.height,
				canvasDisplay: this.canvas.style.display,
				ctxAvailable: !!this.ctx
			});
			
			// 验证bitmap坐标的合理性
			if (bitmap.destRight < bitmap.destLeft || bitmap.destBottom < bitmap.destTop) {
				console.error('[canvas.js] 无效的bitmap坐标:', {
					destLeft: bitmap.destLeft,
					destTop: bitmap.destTop,
					destRight: bitmap.destRight,
					destBottom: bitmap.destBottom
				});
				return;
			}
			
			// 添加数据内容验证
			if (bitmap.data && bitmap.data.length > 0) {
				var sampleData = [];
				var sampleCount = Math.min(20, bitmap.data.length);
				for (var i = 0; i < sampleCount; i++) {
					sampleData.push(bitmap.data[i]);
				}
				console.log('[canvas.js] 原始数据样本 (前' + sampleCount + '字节):', sampleData);
				
				// 检查数据是否全为0或全为255
				var allZero = sampleData.every(function(val) { return val === 0; });
				var all255 = sampleData.every(function(val) { return val === 255; });
				
				if (allZero) {
					console.warn('[canvas.js] ⚠️ 警告：原始数据样本全为0，可能导致全黑显示');
				} else if (all255) {
					console.warn('[canvas.js] ⚠️ 警告：原始数据样本全为255，可能导致全白显示');
				} else {
					console.log('[canvas.js] ✅ 原始数据样本正常');
				}
			}
			
			var output = null;
			if (bitmap.isCompress) {
				console.log('[canvas.js] 检测到压缩数据，开始解压缩...');
				output = decompress(bitmap);
				if (output === null) {
					console.error('[canvas.js] 解压缩失败');
					return;
				}
				console.log('[canvas.js] 解压缩成功，输出数据:', {
					width: output.width,
					height: output.height,
					dataLength: output.data ? output.data.length : 0
				});
			} else {
				console.log('[canvas.js] 处理未压缩数据...');
				output = reverse(bitmap);
				if (output === null) {
					console.error('[canvas.js] 处理未压缩数据失败');
					return;
				}
				console.log('[canvas.js] 未压缩数据处理成功，输出数据:', {
					width: output.width,
					height: output.height,
					dataLength: output.data ? output.data.length : 0
				});
			}
			
			if (!output || !output.data || output.data.length === 0) {
				console.error('[canvas.js] 没有输出数据可以渲染');
				return;
			}
			
			// 验证输出数据内容
			if (output.data && output.data.length > 0) {
				var outputSample = [];
				var outputSampleCount = Math.min(20, output.data.length);
				for (var i = 0; i < outputSampleCount; i++) {
					outputSample.push(output.data[i]);
				}
				console.log('[canvas.js] 输出数据样本 (前' + outputSampleCount + '字节):', outputSample);
				
				// 检查输出数据是否全为0或全为255
				var outputAllZero = outputSample.every(function(val) { return val === 0; });
				var outputAll255 = outputSample.every(function(val) { return val === 255; });
				
				if (outputAllZero) {
					console.error('[canvas.js] ❌ 错误：输出数据样本全为0，将导致全黑显示');
					console.error('[canvas.js] 这可能是数据转换问题，请检查后端颜色转换函数');
				} else if (outputAll255) {
					console.error('[canvas.js] ❌ 错误：输出数据样本全为255，将导致全白显示');
					console.error('[canvas.js] 这可能是数据转换问题，请检查后端颜色转换函数');
				} else {
					console.log('[canvas.js] ✅ 输出数据样本正常');
				}
			}
			
			try {
				console.log('[canvas.js] 开始验证输出数据...');
				
				// 验证输出尺寸
				if (output.width <= 0 || output.height <= 0) {
					console.error('[canvas.js] 输出尺寸无效:', output.width, 'x', output.height);
					return;
				}
				
				// 验证数据长度
				var expectedDataLength = output.width * output.height * 4; // RGBA
				console.log('[canvas.js] 数据长度验证:', {
					expectedLength: expectedDataLength,
					actualLength: output.data.length,
					match: output.data.length === expectedDataLength
				});
				
				if (output.data.length !== expectedDataLength) {
					console.error('[canvas.js] 输出数据长度不匹配。期望:', expectedDataLength, '实际:', output.data.length);
					
					// 添加更详细的调试信息
					console.error('[canvas.js] 最终输出尺寸详情:', {
						outputWidth: output.width,
						outputHeight: output.height,
						expectedBytesPerPixel: 4,
						expectedTotalPixels: output.width * output.height,
						actualBytes: output.data.length,
						actualPixels: output.data.length / 4
					});
					
					// 如果数据长度是合理的倍数，尝试调整输出尺寸
					if (output.data.length > 0 && output.data.length % 4 === 0) {
						var actualPixels = output.data.length / 4;
						var inferredHeight = Math.floor(actualPixels / output.width);
						var inferredWidth = Math.floor(actualPixels / output.height);
						
						console.log('[canvas.js] 尝试推断尺寸:', {
							actualPixels: actualPixels,
							inferredHeight: inferredHeight,
							inferredWidth: inferredWidth
						});
						
						// 优先调整高度
						if (inferredHeight > 0 && inferredHeight <= output.height * 3 && inferredHeight >= output.height * 0.3) {
							console.log('[canvas.js] 调整最终输出高度从', output.height, '到', inferredHeight);
							output.height = inferredHeight;
							expectedDataLength = output.width * output.height * 4;
						} else if (inferredWidth > 0 && inferredWidth <= output.width * 3 && inferredWidth >= output.width * 0.3) {
							console.log('[canvas.js] 调整最终输出宽度从', output.width, '到', inferredWidth);
							output.width = inferredWidth;
							expectedDataLength = output.width * output.height * 4;
						} else {
							// 尝试找到最接近的合理尺寸
							var bestWidth = output.width;
							var bestHeight = output.height;
							var minDiff = Math.abs(output.data.length - expectedDataLength);
							
							var searchRange = Math.max(20, Math.floor(output.width * 0.3));
							for (var w = Math.max(1, output.width - searchRange); w <= output.width + searchRange; w++) {
								for (var h = Math.max(1, output.height - searchRange); h <= output.height + searchRange; h++) {
									var testLength = w * h * 4;
									var diff = Math.abs(output.data.length - testLength);
									if (diff < minDiff) {
										minDiff = diff;
										bestWidth = w;
										bestHeight = h;
									}
								}
							}
							
							if (minDiff < Math.abs(output.data.length - expectedDataLength)) {
								console.log('[canvas.js] 使用最佳匹配最终尺寸:', bestWidth, 'x', bestHeight, '(差异:', minDiff, '字节)');
								output.width = bestWidth;
								output.height = bestHeight;
								expectedDataLength = output.width * output.height * 4;
							}
						}
					}
					
					// 如果仍然不匹配，尝试截断或扩展数据
					if (output.data.length !== expectedDataLength) {
						if (output.data.length > expectedDataLength) {
							console.warn('[canvas.js] 截断输出数据从', output.data.length, '到', expectedDataLength);
							output.data = output.data.slice(0, expectedDataLength);
						} else {
							console.warn('[canvas.js] 输出数据太短，无法渲染');
							return;
						}
					}
				}
				
				console.log('[canvas.js] 创建ImageData对象:', output.width, 'x', output.height);
				
				// use image data to use asm.js
				var imageData = this.ctx.createImageData(output.width, output.height);
				console.log('[canvas.js] ImageData创建成功:', {
					imageDataWidth: imageData.width,
					imageDataHeight: imageData.height,
					imageDataLength: imageData.data.length
				});
				
				console.log('[canvas.js] 设置ImageData数据...');
				imageData.data.set(output.data);
				console.log('[canvas.js] ImageData数据设置完成');
				
				// 修复：使用正确的渲染位置计算
				var renderX = bitmap.destLeft;
				var renderY = bitmap.destTop;
				
				// 验证渲染位置是否在canvas范围内
				console.log('[canvas.js] 渲染位置检查:', {
					renderX: renderX,
					renderY: renderY,
					canvasWidth: this.canvas.width,
					canvasHeight: this.canvas.height,
					imageWidth: output.width,
					imageHeight: output.height,
					xInRange: renderX >= 0 && renderX + output.width <= this.canvas.width,
					yInRange: renderY >= 0 && renderY + output.height <= this.canvas.height,
					expectedWidth: bitmap.destRight - bitmap.destLeft + 1,
					expectedHeight: bitmap.destBottom - bitmap.destTop + 1
				});
				
				// 检查尺寸是否匹配
				var expectedWidth = bitmap.destRight - bitmap.destLeft + 1;
				var expectedHeight = bitmap.destBottom - bitmap.destTop + 1;
				
				if (output.width !== expectedWidth || output.height !== expectedHeight) {
					console.warn('[canvas.js] 输出尺寸与期望尺寸不匹配:', {
						outputWidth: output.width,
						outputHeight: output.height,
						expectedWidth: expectedWidth,
						expectedHeight: expectedHeight,
						widthDiff: output.width - expectedWidth,
						heightDiff: output.height - expectedHeight
					});
					
					// 如果尺寸差异太大，可能需要调整渲染位置
					if (Math.abs(output.width - expectedWidth) > 10 || Math.abs(output.height - expectedHeight) > 10) {
						console.warn('[canvas.js] 尺寸差异较大，可能需要调整渲染策略');
					}
				}
				
				// 检查是否超出canvas边界
				if (renderX < 0 || renderY < 0 || renderX + output.width > this.canvas.width || renderY + output.height > this.canvas.height) {
					console.warn('[canvas.js] 渲染区域超出canvas边界，进行裁剪');
					console.log('[canvas.js] 边界检查详情:', {
						renderX: renderX,
						renderY: renderY,
						outputWidth: output.width,
						outputHeight: output.height,
						canvasWidth: this.canvas.width,
						canvasHeight: this.canvas.height,
						xOverflow: renderX < 0 || renderX + output.width > this.canvas.width,
						yOverflow: renderY < 0 || renderY + output.height > this.canvas.height
					});
					
					// 计算裁剪区域 - 修复裁剪逻辑
					var clipX = Math.max(0, renderX);
					var clipY = Math.max(0, renderY);
					var clipWidth = Math.min(output.width, this.canvas.width - clipX);
					var clipHeight = Math.min(output.height, this.canvas.height - clipY);
					
					// 确保裁剪尺寸不为负数
					clipWidth = Math.max(0, clipWidth);
					clipHeight = Math.max(0, clipHeight);
					
					console.log('[canvas.js] 裁剪计算:', {
						clipX: clipX,
						clipY: clipY,
						clipWidth: clipWidth,
						clipHeight: clipHeight,
						validClip: clipWidth > 0 && clipHeight > 0
					});
					
					if (clipWidth > 0 && clipHeight > 0) {
						// 创建裁剪后的ImageData
						var clippedImageData = this.ctx.createImageData(clipWidth, clipHeight);
						var sourceOffsetX = Math.max(0, -renderX);
						var sourceOffsetY = Math.max(0, -renderY);
						
						console.log('[canvas.js] 裁剪偏移:', {
							sourceOffsetX: sourceOffsetX,
							sourceOffsetY: sourceOffsetY
						});
						
						// 复制数据
						for (var y = 0; y < clipHeight; y++) {
							for (var x = 0; x < clipWidth; x++) {
								var sourceIndex = ((sourceOffsetY + y) * output.width + (sourceOffsetX + x)) * 4;
								var destIndex = (y * clipWidth + x) * 4;
								
								if (sourceIndex < imageData.data.length && destIndex < clippedImageData.data.length) {
									clippedImageData.data[destIndex] = imageData.data[sourceIndex];
									clippedImageData.data[destIndex + 1] = imageData.data[sourceIndex + 1];
									clippedImageData.data[destIndex + 2] = imageData.data[sourceIndex + 2];
									clippedImageData.data[destIndex + 3] = imageData.data[sourceIndex + 3];
								}
							}
						}
						
						console.log('[canvas.js] 开始渲染裁剪后的数据到canvas，位置:', clipX, clipY);
						this.ctx.putImageData(clippedImageData, clipX, clipY);
					} else {
						console.warn('[canvas.js] 裁剪后区域无效，尝试调整渲染位置');
						console.warn('[canvas.js] 裁剪区域详情:', {
							clipWidth: clipWidth,
							clipHeight: clipHeight,
							renderX: renderX,
							renderY: renderY,
							outputWidth: output.width,
							outputHeight: output.height,
							canvasWidth: this.canvas.width,
							canvasHeight: this.canvas.height
						});
						
						// 尝试调整渲染位置到canvas内部
						var adjustedX = Math.max(0, Math.min(renderX, this.canvas.width - output.width));
						var adjustedY = Math.max(0, Math.min(renderY, this.canvas.height - output.height));
						
						console.log('[canvas.js] 尝试调整渲染位置:', {
							originalX: renderX,
							originalY: renderY,
							adjustedX: adjustedX,
							adjustedY: adjustedY
						});
						
						// 如果调整后的位置有效，尝试渲染
						if (adjustedX >= 0 && adjustedY >= 0 && 
							adjustedX + output.width <= this.canvas.width && 
							adjustedY + output.height <= this.canvas.height) {
							console.log('[canvas.js] 使用调整后的位置渲染:', adjustedX, adjustedY);
							this.ctx.putImageData(imageData, adjustedX, adjustedY);
						} else {
							console.error('[canvas.js] 调整后的位置仍然无效，跳过渲染');
							console.error('[canvas.js] 调整后位置检查:', {
								adjustedX: adjustedX,
								adjustedY: adjustedY,
								outputWidth: output.width,
								outputHeight: output.height,
								canvasWidth: this.canvas.width,
								canvasHeight: this.canvas.height,
								xValid: adjustedX >= 0 && adjustedX + output.width <= this.canvas.width,
								yValid: adjustedY >= 0 && adjustedY + output.height <= this.canvas.height
							});
							return;
						}
					}
				} else {
					console.log('[canvas.js] 开始渲染到canvas，位置:', renderX, renderY);
					this.ctx.putImageData(imageData, renderX, renderY);
				}
				
				console.log('[canvas.js] Canvas渲染完成！');
				
				// 更新最后更新时间
				this.lastUpdateTime = Math.floor(Date.now() / 1000); // 修改为秒级时间戳，与后端对齐
				
				// 验证渲染结果
				console.log('[canvas.js] 渲染验证:', {
					canvasDisplay: this.canvas.style.display,
					canvasVisible: this.canvas.style.display !== 'none',
					canvasSize: this.canvas.width + 'x' + this.canvas.height
				});
				
				console.log('=== [canvas.js] bitmap更新处理完成 ===');
			} catch (e) {
				console.error('[canvas.js] 渲染bitmap失败:', e);
				console.error('[canvas.js] 错误堆栈:', e.stack);
			}
		}
	}
	
	/**
	 * Module export
	 */
	Mstsc.Canvas = {
		create : function (canvas) {
			return new Canvas(canvas);
		}
	}
	
	// 导出解压缩函数供外部使用
	Mstsc.decompress = function(bitmap) {
		console.log('[canvas.js] Mstsc.decompress called for bitmap:', {
			bitsPerPixel: bitmap.bitsPerPixel,
			width: bitmap.width,
			height: bitmap.height,
			isCompress: bitmap.isCompress,
			dataLength: bitmap.data ? bitmap.data.length : 0
		});
		
		// 基本验证
		if (!bitmap.data || bitmap.data.length === 0) {
			console.error('[canvas.js] No bitmap data provided to Mstsc.decompress');
			return null;
		}
		
		if (bitmap.width <= 0 || bitmap.height <= 0) {
			console.error('[canvas.js] Invalid bitmap dimensions in Mstsc.decompress:', bitmap.width, 'x', bitmap.height);
			return null;
		}
		
		// 如果标记为未压缩，直接返回数据
		if (!bitmap.isCompress) {
			console.log('[canvas.js] Bitmap is not compressed, returning original data');
			return bitmap.data;
		}
		
		// 尝试解压缩
		var result = decompress(bitmap);
		if (result && result.data) {
			console.log('[canvas.js] Decompression successful, returning', result.data.length, 'bytes');
			return result.data;
		} else {
			console.error('[canvas.js] Decompression failed');
			
			// 尝试将数据作为未压缩数据处理
			var expectedUncompressedLength = bitmap.width * bitmap.height * (bitmap.bitsPerPixel / 8);
			if (bitmap.data && bitmap.data.length === expectedUncompressedLength) {
				console.log('[canvas.js] 数据长度匹配未压缩数据，尝试作为未压缩数据处理');
				return bitmap.data;
			}
			
			// 如果数据长度不匹配，但仍然尝试处理
			console.log('[canvas.js] 尝试处理可能损坏的压缩数据');
			var fallbackResult = processUncompressedData(bitmap);
			if (fallbackResult && fallbackResult.data) {
				console.log('[canvas.js] 回退处理成功，返回', fallbackResult.data.length, '字节');
				return fallbackResult.data;
			}
			
			// 显示具体的错误信息
			if (typeof showDecompressError === 'function') {
				showDecompressError('位图解压缩失败，请检查数据格式');
			}
			return null;
		}
	};
})();
