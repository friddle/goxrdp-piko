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
	 * Mouse button mapping
	 * @param button {integer} client button number
	 */
	function mouseButtonMap(button) {
		// 使用标准RDP协议按钮编号
		// 浏览器按钮编号：0=左键，1=中键，2=右键
		// RDP协议按钮编号：0=左键，1=中键，2=右键
		// 直接透传，保持一致性
		return button;
	};
	
	/**
	 * 检查是否为拖拽操作
	 * @param button {integer} 鼠标按钮编号
	 * @param pressedButtons {Set} 当前按下的按钮集合
	 * @returns {boolean} 是否为拖拽操作
	 */
	function isDragOperation(button, pressedButtons) {
		// 只有左键（button=0）才允许拖拽操作
		// 右键（button=2）和中键（button=1）不进行拖拽
		return button === 0 && pressedButtons.has(0);
	}
	
	function getCanvasRelativePosition(e, canvas) {
		var rect = canvas.getBoundingClientRect();
		var scaleX = canvas.width / rect.width;
		var scaleY = canvas.height / rect.height;
		
		// 计算相对于canvas的坐标
		var x = (e.clientX - rect.left) * scaleX;
		var y = (e.clientY - rect.top) * scaleY;
		
		// 确保坐标在有效范围内
		x = Math.max(0, Math.min(canvas.width - 1, x));
		y = Math.max(0, Math.min(canvas.height - 1, y));
		
		return {
			x: Math.round(x),
			y: Math.round(y)
		};
	}
	
	/**
	 * 修复Canvas尺寸和坐标缩放
	 */
	function fixCanvasScaling(canvas) {
		if (!canvas) return false;
		
		// 获取canvas的显示尺寸
		var displayWidth = canvas.clientWidth || canvas.offsetWidth;
		var displayHeight = canvas.clientHeight || canvas.offsetHeight;
		
		// 如果CSS尺寸未设置，使用窗口尺寸
		if (!displayWidth || !displayHeight) {
			displayWidth = window.innerWidth;
			displayHeight = window.innerHeight;
			
			// 设置CSS尺寸
			canvas.style.width = displayWidth + 'px';
			canvas.style.height = displayHeight + 'px';
		}
		
		// 确保canvas像素尺寸与显示尺寸一致
		if (canvas.width !== displayWidth || canvas.height !== displayHeight) {
			canvas.width = displayWidth;
			canvas.height = displayHeight;
			console.log('[client.js] Canvas尺寸已更新:', displayWidth, 'x', displayHeight);
		}
		
		return true;
	}
	
	/**
	 * 检查并修复连接状态
	 */
	function checkAndFixConnection(self) {
		// console.log('检查连接状态:', {
		// 	hasSocket: !!self.socket,
		// 	socketState: self.socket ? self.socket.readyState : 'no socket',
		// 	activeSession: self.activeSession,
		// 	windowWs: typeof window.ws,
		// 	windowWsState: window.ws ? window.ws.readyState : 'undefined'
		// });
		
		// 如果socket不存在或已关闭，尝试重新获取
		if (!self.socket || self.socket.readyState !== WebSocket.OPEN) {
			if (typeof window.ws !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
				self.socket = window.ws;
				return true;
			} else {
				return false;
			}
		}
		
		// 如果activeSession为false但socket可用，尝试强制激活
		if (!self.activeSession && self.socket && self.socket.readyState === WebSocket.OPEN) {
			self.forceActivateSession();
		}
		
		return true;
	}
	
	/**
	 * 检测数据是否全黑
	 */
	function isAllBlackData(data, sampleCount) {
		if (!data || data.length === 0) return true;
		
		sampleCount = sampleCount || Math.min(100, data.length);
		var allBlack = true;
		var nonZeroCount = 0;
		var totalChecked = 0;
		
		for (var i = 0; i < sampleCount; i += 4) { // 每4字节一个像素
			if (i + 2 < data.length) {
				totalChecked++;
				// 检查RGB通道是否都为0（忽略Alpha通道）
				if (data[i] !== 0 || data[i+1] !== 0 || data[i+2] !== 0) {
					nonZeroCount++;
					// 如果发现非零像素，检查是否足够多
					if (nonZeroCount > 10) { // 允许最多10个非零像素
						allBlack = false;
						break;
					}
				}
			}
		}
		
		// 添加调试信息
		if (allBlack && totalChecked > 0) {
			console.log('[client.js] 全黑检测详情:', {
				totalChecked: totalChecked,
				nonZeroCount: nonZeroCount,
				allBlack: allBlack,
				firstFewPixels: [
					data[0], data[1], data[2], data[3],
					data[4], data[5], data[6], data[7],
					data[8], data[9], data[10], data[11]
				]
			});
		}
		
		return allBlack;
	}
	
	/**
	 * 处理Base64编码的图像数据
	 */
	function processBase64ImageData(base64Data, width, height) {
		try {
			// 将Base64字符串转换为二进制数据
			var binaryString = atob(base64Data);
			var decodedData = new Uint8Array(binaryString.length);
			for (var k = 0; k < binaryString.length; k++) {
				decodedData[k] = binaryString.charCodeAt(k);
			}
			
			// 添加调试信息
			console.log('[client.js] Base64解码详情:', {
				base64Length: base64Data.length,
				binaryLength: binaryString.length,
				decodedLength: decodedData.length,
				expectedLength: width * height * 4,
				firstFewBytes: [
					decodedData[0], decodedData[1], decodedData[2], decodedData[3],
					decodedData[4], decodedData[5], decodedData[6], decodedData[7]
				]
			});
			
			// 检测解码后的全黑数据
			if (decodedData.length > 0 && isAllBlackData(decodedData, 100)) {
				console.warn('[client.js] ⚠️ 警告：解码后数据全黑，跳过渲染');
				return null; // 返回null表示跳过渲染
			}
			
			return decodedData;
		} catch (e) {
			console.error('[client.js] Base64解码失败:', e);
			return null;
		}
	}
	
	/**
	 * Mstsc client
	 * Input client connection (mouse and keyboard)
	 * bitmap processing
	 * @param canvas {canvas} rendering element
	 */
	function Client(canvas) {
		this.canvas = canvas;
		// create renderer
		this.render = new Mstsc.Canvas.create(this.canvas); 
		this.socket = null;
		this.activeSession = false;
		
		// 添加鼠标状态跟踪
		this.mouseState = {
			isDragging: false,
			lastX: 0,
			lastY: 0,
			pressedButtons: new Set(),
			lastMoveTime: 0
		};
		
		// 添加窗口大小变化监听
		var self = this;
		window.addEventListener('resize', function() {
			setTimeout(function() {
				fixCanvasScaling(self.canvas);
			}, 100);
		});
		
		this.install();
	}
	
	Client.prototype = {
		install : function () {
			var self = this;
			
			// 初始化时修复Canvas缩放
			fixCanvasScaling(this.canvas);
			
			// bind mouse move event
			this.canvas.addEventListener('mousemove', function (e) {
				// 检查并尝试修复连接状态
				if (!checkAndFixConnection(self)) {
					return;
				}
				
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
					return;
				}
				
				var pos = getCanvasRelativePosition(e, self.canvas);
				
				// 降低采样率：每50ms才发送一次鼠标移动事件
				var now = Date.now();
				if (now - self.mouseState.lastMoveTime < 50) {
					return;
				}
				self.mouseState.lastMoveTime = now;
				
				// 更新鼠标状态
				self.mouseState.lastX = pos.x;
				self.mouseState.lastY = pos.y;
				
				// 简化：直接发送鼠标移动事件，让后端处理拖拽逻辑
				var mouseEvent = {
					event: 'mouse',
					data: [pos.x, pos.y, 0, false] // 移动事件，按钮状态为false
				};
				self.socket.send(JSON.stringify(mouseEvent));
				
				e.preventDefault();
				return false;
			});
			
			this.canvas.addEventListener('mousedown', function (e) {
				// 检查并尝试修复连接状态
				if (!checkAndFixConnection(self)) {
					return;
				}
				
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
					return;
				}
				
				var pos = getCanvasRelativePosition(e, self.canvas);
				var mappedButton = mouseButtonMap(e.button);
				
				// 更新鼠标状态
				self.mouseState.pressedButtons.add(mappedButton);
				self.mouseState.lastX = pos.x;
				self.mouseState.lastY = pos.y;
				
				// 简化：直接发送鼠标按下事件
				var mouseEvent = {
					event: 'mouse',
					data: [pos.x, pos.y, mappedButton, true]
				};
				
				try {
					self.socket.send(JSON.stringify(mouseEvent));
				} catch (error) {
				}
				
				e.preventDefault();
				return false;
			});
			
			// 原有canvas mouseup事件监听
			this.canvas.addEventListener('mouseup', function (e) {
				// 检查并尝试修复连接状态
				if (!checkAndFixConnection(self)) {
					return;
				}
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
					return;
				}
				var pos = getCanvasRelativePosition(e, self.canvas);
				var mappedButton = mouseButtonMap(e.button);
				
				// 更新鼠标状态
				self.mouseState.pressedButtons.delete(mappedButton);
				
				// 简化：直接发送鼠标释放事件
				var mouseEvent = {
					event: 'mouse',
					data: [pos.x, pos.y, mappedButton, false]
				};
				try {
					self.socket.send(JSON.stringify(mouseEvent));
				} catch (error) {
				}
				e.preventDefault();
				return false;
			});
			
			// 新增window mouseup事件监听
			window.addEventListener('mouseup', function (e) {
				// 检查并尝试修复连接状态
				if (!checkAndFixConnection(self)) {
					return;
				}
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
					return;
				}
				var pos = getCanvasRelativePosition(e, self.canvas);
				var mappedButton = mouseButtonMap(e.button);
				
				// 更新鼠标状态
				self.mouseState.pressedButtons.delete(mappedButton);
				
				// 简化：直接发送鼠标释放事件
				var mouseEvent = {
					event: 'mouse',
					data: [pos.x, pos.y, mappedButton, false]
				};
				try {
					self.socket.send(JSON.stringify(mouseEvent));
				} catch (error) {
				}
				e.preventDefault();
				return false;
			});
			
			this.canvas.addEventListener('contextmenu', function (e) {
				// 检查并尝试修复连接状态
				if (!checkAndFixConnection(self)) {
					return;
				}
				
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
					return;
				}
				
				var pos = getCanvasRelativePosition(e, self.canvas);
				var mouseEvent = {
					event: 'mouse',
					data: [pos.x, pos.y, mouseButtonMap(e.button), false]
				};
				
				try {
					self.socket.send(JSON.stringify(mouseEvent));
				} catch (error) {
				}
				
				e.preventDefault();
				return false;
			});
			this.canvas.addEventListener('DOMMouseScroll', function (e) {
				// 检查并尝试修复连接状态
				if (!checkAndFixConnection(self)) {
					return;
				}
				
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
					return;
				}
				
				var isHorizontal = false;
				var delta = e.detail;
				var step = Math.round(Math.abs(delta) * 15 / 8);
				var pos = getCanvasRelativePosition(e, self.canvas);
				var wheelEvent = {
					event: 'wheel',
					data: [pos.x, pos.y, step, delta > 0, isHorizontal]
				};
				
				try {
					self.socket.send(JSON.stringify(wheelEvent));
				} catch (error) {
				}
				
				e.preventDefault();
				return false;
			});
			this.canvas.addEventListener('mousewheel', function (e) {
				// 检查并尝试修复连接状态
				if (!checkAndFixConnection(self)) {
					return;
				}
				
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
					return;
				}
				
				var isHorizontal = Math.abs(e.deltaX) > Math.abs(e.deltaY);
				var delta = isHorizontal ? e.deltaX : e.deltaY;
				var step = Math.round(Math.abs(delta) * 15 / 8);
				var pos = getCanvasRelativePosition(e, self.canvas);
				var wheelEvent = {
					event: 'wheel',
					data: [pos.x, pos.y, step, delta > 0, isHorizontal]
				};
				
				try {
					self.socket.send(JSON.stringify(wheelEvent));
				} catch (error) {
				}
				
				e.preventDefault();
				return false;
			});
			
			// bind keyboard event
			window.addEventListener('keydown', function (e) {
					// console.log('检查连接状态:', {
					// 	key: e.key,
					// 	code: e.code,
					// 	keyCode: e.keyCode,
					// 	socketReady: self.socket && self.socket.readyState === WebSocket.OPEN,
					// 	activeSession: self.activeSession,
					// 	scancode: Mstsc.scancode(e)
					// });
					
				// 检查并尝试修复连接状态
				if (!self.checkAndFixConnection()) {
					return;
				}
				
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
						// console.log('检查连接状态:', {
						// 	hasSocket: !!self.socket,
						// 	socketState: self.socket ? self.socket.readyState : 'no socket',
						// 	activeSession: self.activeSession
						// });
					
					// 如果activeSession为false，尝试强制激活（用于调试）
					if (!self.activeSession && self.socket && self.socket.readyState === WebSocket.OPEN) {
						self.forceActivateSession();
					}
					return;
				}
				
				var scancode = Mstsc.scancode(e);
				if (!scancode || scancode === 0) {
					return;
				}
				
				var keyboardEvent = {
					event: 'scancode',
					data: [scancode, true]
				};
				self.socket.send(JSON.stringify(keyboardEvent));

				e.preventDefault();
				return false;
			});
			window.addEventListener('keyup', function (e) {
					// console.log('检查连接状态:', {
					// 	key: e.key,
					// 	code: e.code,
					// 	keyCode: e.keyCode,
					// 	socketReady: self.socket && self.socket.readyState === WebSocket.OPEN,
					// 	activeSession: self.activeSession,
					// 	scancode: Mstsc.scancode(e)
					// });
					
				// 检查并尝试修复连接状态
				if (!self.checkAndFixConnection()) {
					return;
				}
				
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
						// console.log('检查连接状态:', {
						// 	hasSocket: !!self.socket,
						// 	socketState: self.socket ? self.socket.readyState : 'no socket',
						// 	activeSession: self.activeSession
						// });
					
					// 如果activeSession为false，尝试强制激活（用于调试）
					if (!self.activeSession && self.socket && self.socket.readyState === WebSocket.OPEN) {
						self.forceActivateSession();
					}
					return;
				}
				
				var scancode = Mstsc.scancode(e);
				if (!scancode || scancode === 0) {
					return;
				}
				
				var keyboardEvent = {
					event: 'scancode',
					data: [scancode, false]
				};
				self.socket.send(JSON.stringify(keyboardEvent));
				
				e.preventDefault();
				return false;
			});
			
			return this;
		},
		/**
		 * connect
		 * @param ip {string} ip target for rdp
		 * @param domain {string} microsoft domain
		 * @param username {string} session username
		 * @param password {string} session password
		 * @param next {function} asynchrone end callback
		 */
		connect : function (ip, domain, username, password, next) {
			var self = this;
			
				// console.log('检查连接状态:', {
				// 	ip: ip,
				// 	domain: domain,
				// 	username: username,
				// 	hasPassword: !!password,
				// 	windowWs: typeof window.ws,
				// 	windowWsState: window.ws ? window.ws.readyState : 'undefined'
				// });
			
			// 检查是否已经有全局的WebSocket连接
			if (typeof window.ws !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
				this.socket = window.ws;
				
				// 设置消息处理器
				this.setupMessageHandler(next);
				
				// 发送连接信息
				this.sendConnectionInfo(ip, domain, username, password);
			} else {
				
				// 等待WebSocket连接建立
				var checkConnection = function() {
						// console.log('检查连接状态:', {
						// 	windowWs: typeof window.ws,
						// 	windowWsState: window.ws ? window.ws.readyState : 'undefined',
						// 	windowWsOpen: window.ws ? window.ws.readyState === WebSocket.OPEN : false
						// });
					
					if (typeof window.ws !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
						self.socket = window.ws;
						
						// 设置消息处理器
						self.setupMessageHandler(next);
						
						// 发送连接信息
						self.sendConnectionInfo(ip, domain, username, password);
					} else {
						setTimeout(checkConnection, 100);
					}
				};
				
				checkConnection();
			}
		},
		
		/**
		 * 设置消息处理器
		 */
		setupMessageHandler : function(next) {
			var self = this;
			
				// console.log('检查连接状态:', {
				// 	hasSocket: !!this.socket,
				// 	socketState: this.socket ? this.socket.readyState : 'no socket',
				// 	activeSession: this.activeSession
				// });
			
			// 保存原有的消息处理器
			var originalOnMessage = this.socket.onmessage;
			
			this.socket.onmessage = function(event) {
				try {
					var message = JSON.parse(event.data);
					
					switch(message.event) {
						case 'rdp-connect':
							if (message.data && message.data.reused) {
								// 可以在这里添加复用连接的提示信息
								if (typeof window.showReusedConnectionMessage === 'function') {
									window.showReusedConnectionMessage(message.data.message || '复用现有RDP连接');
								}
							}
							self.activeSession = true;
							
							// 连接成功后发送分辨率更新
							setTimeout(function() {
								self.sendResolutionUpdate();
							}, 500);
							break;
						case 'rdp-bitmap':
							// 	bitsPerPixel: message.data.bitsPerPixel,
							// 	rectanglesCount: message.data.rectangles ? message.data.rectangles.length : 0,
							// 	timestamp: message.data.timestamp,
							// 	hasRectangles: !!message.data.rectangles,
							// 	isArray: Array.isArray(message.data.rectangles)
							// });
							
							// 处理多个矩形数据
							if (message.data.rectangles && Array.isArray(message.data.rectangles)) {
								
								message.data.rectangles.forEach(function(rect, index) {
									// 	destLeft: rect.destLeft,
									// 	destTop: rect.destTop,
									// 	destRight: rect.destRight,
									// 	destBottom: rect.destBottom,
									// 	width: rect.width,
									// 	height: rect.height,
									// 	bitsPerPixel: rect.bitsPerPixel,
									// 	isCompress: rect.isCompress,
									// 	dataLength: rect.data ? rect.data.length : 0,
									// 	dataType: rect.data ? rect.data.constructor.name : 'undefined'
									// });
									
									// 验证矩形数据
									if (!rect.data || rect.data.length === 0) {
										return;
									}
									
									if (rect.width <= 0 || rect.height <= 0) {
										return;
									}
									
									if (rect.destRight < rect.destLeft || rect.destBottom < rect.destTop) {
										return;
									}
									
									// 后端已经解压缩，前端直接渲染
									
									// 处理Base64编码的数据
									var decodedData = null;
									if (typeof rect.data === 'string') {
										// 使用新的处理函数
										decodedData = processBase64ImageData(rect.data, rect.width, rect.height);
										if (decodedData === null) {
											// 全黑数据，跳过渲染
											return;
										}
									} else {
										// 如果不是字符串，直接使用原始数据
										decodedData = rect.data;
										
										// 检查原始数据是否全黑
										if (isAllBlackData(decodedData, 100)) {
											console.warn('[client.js] ⚠️ 警告：原始数据全黑，跳过渲染');
											return;
										}
									}
									
									// 验证数据长度是否正确（RGBA格式，4字节/像素）
									var expectedDataLength = rect.width * rect.height * 4;
									if (decodedData.length !== expectedDataLength) {
										
										// 添加详细的数据类型调试信息
										// 	dataType: typeof decodedData,
										// 	constructor: decodedData ? decodedData.constructor.name : 'undefined',
										// 	isArray: Array.isArray(decodedData),
										// 	isUint8Array: decodedData instanceof Uint8Array,
										// 	isArrayBuffer: decodedData instanceof ArrayBuffer,
										// 	hasSlice: decodedData && typeof decodedData.slice === 'function',
										// 	hasLength: decodedData && typeof decodedData.length !== 'undefined'
										// });
										
										// 如果数据长度不匹配，尝试调整
										if (decodedData.length < expectedDataLength) {
											return;
										}
										// 如果数据过多，截断到正确长度
										if (decodedData.length > expectedDataLength) {
											// 修复：使用正确的方式截断数据，避免数据丢失
											var originalData = decodedData;
											var truncatedData = null;
											
											if (decodedData instanceof Uint8Array) {
												truncatedData = decodedData.slice(0, expectedDataLength);
											} else if (decodedData instanceof ArrayBuffer) {
												truncatedData = decodedData.slice(0, expectedDataLength);
											} else if (Array.isArray(decodedData)) {
												truncatedData = decodedData.slice(0, expectedDataLength);
											} else if (decodedData && typeof decodedData.slice === 'function') {
												truncatedData = decodedData.slice(0, expectedDataLength);
											} else {
												// 如果是其他类型的数据，尝试转换为Uint8Array
												try {
													var tempArray = new Uint8Array(decodedData);
													truncatedData = tempArray.slice(0, expectedDataLength);
												} catch (e) {
													return;
												}
											}
											
											// 验证截断结果
											if (truncatedData && truncatedData.length === expectedDataLength) {
												// 	originalLength: originalData.length,
												// 	truncatedLength: truncatedData.length,
												// 	expectedLength: expectedDataLength,
												// 	truncatedType: truncatedData.constructor.name
												// });
												decodedData = truncatedData;
											} else {
													// console.log('检查连接状态:', {
													// 	truncatedData: !!truncatedData,
													// 	truncatedLength: truncatedData ? truncatedData.length : 'undefined',
													// 	expectedLength: expectedDataLength
													// });
													return;
											}
										}
									}
									
									// 创建新的矩形对象，使用解码后的数据
									var processedRect = {
										destLeft: rect.destLeft,
										destTop: rect.destTop,
										destRight: rect.destRight,
										destBottom: rect.destBottom,
										width: rect.width,
										height: rect.height,
										bitsPerPixel: rect.bitsPerPixel,
										isCompress: rect.isCompress,
										data: decodedData
									};
									
									try {
										self.render.update(processedRect);
									} catch (e) {
									}
								});
								
							} else {
								
								// 兼容单个位图对象的情况
								try {
									self.render.update(message.data);
								} catch (e) {
								}
							}
							
							break;
						case 'rdp-close':
							next(null);
							self.activeSession = false;
							break;
						case 'rdp-error':
							next(message.data);
							self.activeSession = false;
							break;
					}
					
					// 调用原有的消息处理器（如果存在）
					if (originalOnMessage) {
						originalOnMessage.call(this, event);
					}
				} catch (e) {
				}
			};
		},
		
		/**
		 * 发送连接信息
		 */
		sendConnectionInfo : function(ip, domain, username, password) {
			// 发送连接信息
			var infos = {
				event: 'infos',
				data: {
					ip: ip.indexOf(":")>-1 ? ip.split(":")[0] : ip,
					port: ip.indexOf(":")>-1 ? parseInt(ip.split(":")[1]) : 3389,
					screen: { 
						width: this.canvas.width, 
						height: this.canvas.height 
					}, 
					domain: domain, 
					username: username, 
					password: password, 
					locale: Mstsc.locale()
				}
			};
			
				// console.log('检查连接状态:', {
				// 	ip: infos.data.ip,
				// 	port: infos.data.port,
				// 	screen: infos.data.screen,
				// 	domain: infos.data.domain,
				// 	username: infos.data.username,
				// 	hasPassword: !!infos.data.password
				// });
			
			this.socket.send(JSON.stringify(infos));
		},
		
		/**
		 * 处理位图消息（从 index.html 转发）
		 */
		handleBitmapMessage : function(message) {
			// 	bitsPerPixel: message.data.bitsPerPixel,
			// 	rectanglesCount: message.data.rectangles ? message.data.rectangles.length : 0,
			// 	timestamp: message.data.timestamp,
			// 	hasRectangles: !!message.data.rectangles,
			// 	isArray: Array.isArray(message.data.rectangles)
			// });
			
			// 处理多个矩形数据
			if (message.data.rectangles && Array.isArray(message.data.rectangles)) {
				
				message.data.rectangles.forEach(function(rect, index) {
					// 	destLeft: rect.destLeft,
					// 	destTop: rect.destTop,
					// 	destRight: rect.destRight,
					// 	destBottom: rect.destBottom,
					// 	width: rect.width,
					// 	height: rect.height,
					// 	bitsPerPixel: rect.bitsPerPixel,
					// 	isCompress: rect.isCompress,
					// 	dataLength: rect.data ? rect.data.length : 0,
					// 	dataType: rect.data ? rect.data.constructor.name : 'undefined'
					// });
					
					// 验证矩形数据
					if (!rect.data || rect.data.length === 0) {
						return;
					}
					
					if (rect.width <= 0 || rect.height <= 0) {
						return;
					}
					
					if (rect.destRight < rect.destLeft || rect.destBottom < rect.destTop) {
						return;
					}
					
					// 后端已经解压缩，前端直接渲染
					
					// 处理Base64编码的数据
					var decodedData = null;
					if (typeof rect.data === 'string') {
						// 使用新的处理函数
						decodedData = processBase64ImageData(rect.data, rect.width, rect.height);
						if (decodedData === null) {
							// 全黑数据，跳过渲染
							return;
						}
					} else {
						// 如果不是字符串，直接使用原始数据
						decodedData = rect.data;
						
						// 检查原始数据是否全黑
						if (isAllBlackData(decodedData, 100)) {
							console.warn('[client.js] ⚠️ 警告：原始数据全黑，跳过渲染');
							return;
						}
					}
					
					// 验证数据长度是否正确（RGBA格式，4字节/像素）
					var expectedDataLength = rect.width * rect.height * 4;
					if (decodedData.length !== expectedDataLength) {
						
						// 添加详细的数据类型调试信息
						// 	dataType: typeof decodedData,
						// 	constructor: decodedData ? decodedData.constructor.name : 'undefined',
						// 	isArray: Array.isArray(decodedData),
						// 	isUint8Array: decodedData instanceof Uint8Array,
						// 	isArrayBuffer: decodedData instanceof ArrayBuffer,
						// 	hasSlice: decodedData && typeof decodedData.slice === 'function',
						// 	hasLength: decodedData && typeof decodedData.length !== 'undefined'
						// });
						
						// 如果数据长度不匹配，尝试调整
						if (decodedData.length < expectedDataLength) {
							return;
						}
						// 如果数据过多，截断到正确长度
						if (decodedData.length > expectedDataLength) {
							// 修复：使用正确的方式截断数据，避免数据丢失
							var originalData = decodedData;
							var truncatedData = null;
							
							if (decodedData instanceof Uint8Array) {
								truncatedData = decodedData.slice(0, expectedDataLength);
							} else if (decodedData instanceof ArrayBuffer) {
								truncatedData = decodedData.slice(0, expectedDataLength);
							} else if (Array.isArray(decodedData)) {
								truncatedData = decodedData.slice(0, expectedDataLength);
							} else if (decodedData && typeof decodedData.slice === 'function') {
								truncatedData = decodedData.slice(0, expectedDataLength);
							} else {
								// 如果是其他类型的数据，尝试转换为Uint8Array
								try {
									var tempArray = new Uint8Array(decodedData);
									truncatedData = tempArray.slice(0, expectedDataLength);
								} catch (e) {
									return;
								}
							}
							
							// 验证截断结果
							if (truncatedData && truncatedData.length === expectedDataLength) {
								// 	originalLength: originalData.length,
								// 	truncatedLength: truncatedData.length,
								// 	expectedLength: expectedDataLength,
								// 	truncatedType: truncatedData.constructor.name
								// });
								decodedData = truncatedData;
							} else {
									// console.log('检查连接状态:', {
									// 	truncatedData: !!truncatedData,
									// 	truncatedLength: truncatedData ? truncatedData.length : 'undefined',
									// 	expectedLength: expectedDataLength
									// });
									return;
							}
						}
					}
					
					// 创建新的矩形对象，使用解码后的数据
					var processedRect = {
						destLeft: rect.destLeft,
						destTop: rect.destTop,
						destRight: rect.destRight,
						destBottom: rect.destBottom,
						width: rect.width,
						height: rect.height,
						bitsPerPixel: rect.bitsPerPixel,
						isCompress: rect.isCompress,
						data: decodedData
					};
					
					try {
						this.render.update(processedRect);
					} catch (e) {
					}
				}.bind(this));
				
			} else {
				
				// 兼容单个位图对象的情况
				try {
					this.render.update(message.data);
				} catch (e) {
				}
			}
			
		},
		
		/**
		 * 检查连接状态并尝试修复
		 */
		checkAndFixConnection : function() {
			var self = this;
			
				// console.log('检查连接状态:', {
				// 	hasSocket: !!this.socket,
				// 	socketState: this.socket ? this.socket.readyState : 'no socket',
				// 	activeSession: this.activeSession,
				// 	windowWs: typeof window.ws,
				// 	windowWsState: window.ws ? window.ws.readyState : 'undefined'
				// });
			
			// 如果socket不存在或已关闭，尝试重新获取
			if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
				if (typeof window.ws !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
					this.socket = window.ws;
					return true;
				} else {
					return false;
				}
			}
			
			return true;
		},
		
		/**
		 * 强制激活会话（用于调试）
		 */
		forceActivateSession : function() {
			this.activeSession = true;
		},
		
		/**
		 * 发送分辨率更新信息
		 */
		sendResolutionUpdate : function() {
			if (this.socket && this.socket.readyState === WebSocket.OPEN) {
				var msg = {
					event: 'resolution-update',
					data: {
						width: this.canvas.width,
						height: this.canvas.height,
						displayWidth: this.canvas.clientWidth || this.canvas.offsetWidth,
						displayHeight: this.canvas.clientHeight || this.canvas.offsetHeight
					}
				};
				
				try {
					this.socket.send(JSON.stringify(msg));
				} catch (error) {
				}
			} else {
			}
		}
	}
	
	// 显示解压缩错误信息的函数
	function showDecompressError(message) {
		// 检查是否在浏览器环境中
		if (typeof document !== 'undefined' && typeof window !== 'undefined') {
			// 创建一个错误提示框
			var errorDiv = document.createElement('div');
			errorDiv.style.cssText = 'position: fixed; top: 20px; right: 20px; background: #f8d7da; color: #721c24; padding: 15px; border-radius: 5px; border: 1px solid #f5c6cb; z-index: 1000; max-width: 300px; font-size: 14px;';
			errorDiv.innerHTML = '<strong>解压缩错误:</strong><br>' + message;
			document.body.appendChild(errorDiv);
			
			// 5秒后自动移除提示
			setTimeout(function() {
				if (errorDiv.parentNode) {
					errorDiv.parentNode.removeChild(errorDiv);
				}
			}, 5000);
		}
	}
	
	Mstsc.client = {
		create : function (canvas) {
			return new Client(canvas);
		}
	}
})();
