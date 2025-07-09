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
		switch(button) {
		case 0:
			return 1;
		case 2:
			return 2;
		default:
			return 0;
		}
	};
	
	function getCanvasRelativePosition(e, canvas) {
		var rect = canvas.getBoundingClientRect();
		var scaleX = canvas.width / rect.width;
		var scaleY = canvas.height / rect.height;
		return {
			x: (e.clientX - rect.left) * scaleX,
			y: (e.clientY - rect.top) * scaleY
		};
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
		this.install();
	}
	
	Client.prototype = {
		install : function () {
			var self = this;
			// bind mouse move event
			this.canvas.addEventListener('mousemove', function (e) {
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) return;
				var pos = getCanvasRelativePosition(e, self.canvas);
				var mouseEvent = {
					event: 'mouse',
					data: [Math.round(pos.x), Math.round(pos.y), 0, false]
				};
				self.socket.send(JSON.stringify(mouseEvent));
				e.preventDefault();
				return false;
			});
			this.canvas.addEventListener('mousedown', function (e) {
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) return;
				var pos = getCanvasRelativePosition(e, self.canvas);
				var mouseEvent = {
					event: 'mouse',
					data: [Math.round(pos.x), Math.round(pos.y), mouseButtonMap(e.button), true]
				};
				self.socket.send(JSON.stringify(mouseEvent));
				e.preventDefault();
				return false;
			});
			this.canvas.addEventListener('mouseup', function (e) {
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) return;
				var pos = getCanvasRelativePosition(e, self.canvas);
				var mouseEvent = {
					event: 'mouse',
					data: [Math.round(pos.x), Math.round(pos.y), mouseButtonMap(e.button), false]
				};
				self.socket.send(JSON.stringify(mouseEvent));
				e.preventDefault();
				return false;
			});
			this.canvas.addEventListener('contextmenu', function (e) {
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) return;
				var pos = getCanvasRelativePosition(e, self.canvas);
				var mouseEvent = {
					event: 'mouse',
					data: [Math.round(pos.x), Math.round(pos.y), mouseButtonMap(e.button), false]
				};
				self.socket.send(JSON.stringify(mouseEvent));
				e.preventDefault();
				return false;
			});
			this.canvas.addEventListener('DOMMouseScroll', function (e) {
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) return;
				var isHorizontal = false;
				var delta = e.detail;
				var step = Math.round(Math.abs(delta) * 15 / 8);
				var pos = getCanvasRelativePosition(e, self.canvas);
				var wheelEvent = {
					event: 'wheel',
					data: [Math.round(pos.x), Math.round(pos.y), step, delta > 0, isHorizontal]
				};
				self.socket.send(JSON.stringify(wheelEvent));
				e.preventDefault();
				return false;
			});
			this.canvas.addEventListener('mousewheel', function (e) {
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) return;
				var isHorizontal = Math.abs(e.deltaX) > Math.abs(e.deltaY);
				var delta = isHorizontal?e.deltaX:e.deltaY;
				var step = Math.round(Math.abs(delta) * 15 / 8);
				var pos = getCanvasRelativePosition(e, self.canvas);
				var wheelEvent = {
					event: 'wheel',
					data: [Math.round(pos.x), Math.round(pos.y), step, delta > 0, isHorizontal]
				};
				self.socket.send(JSON.stringify(wheelEvent));
				e.preventDefault();
				return false;
			});
			
			// bind keyboard event
			window.addEventListener('keydown', function (e) {
				console.log('[client.js] 键盘按下事件:', {
					key: e.key,
					code: e.code,
					keyCode: e.keyCode,
					socketReady: self.socket && self.socket.readyState === WebSocket.OPEN,
					activeSession: self.activeSession,
					scancode: Mstsc.scancode(e)
				});
				
				// 检查并尝试修复连接状态
				if (!self.checkAndFixConnection()) {
					console.warn('[client.js] 无法修复连接状态，键盘事件被忽略');
					return;
				}
				
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
					console.warn('[client.js] 键盘事件被忽略:', {
						hasSocket: !!self.socket,
						socketState: self.socket ? self.socket.readyState : 'no socket',
						activeSession: self.activeSession
					});
					
					// 如果activeSession为false，尝试强制激活（用于调试）
					if (!self.activeSession && self.socket && self.socket.readyState === WebSocket.OPEN) {
						console.log('[client.js] 尝试强制激活会话');
						self.forceActivateSession();
					}
					return;
				}
				
				var scancode = Mstsc.scancode(e);
				if (!scancode || scancode === 0) {
					console.warn('[client.js] 无效的扫描码:', scancode, 'for key:', e.key, 'code:', e.code);
					return;
				}
				
				var keyboardEvent = {
					event: 'scancode',
					data: [scancode, true]
				};
				console.log('[client.js] 发送键盘按下事件:', keyboardEvent);
				self.socket.send(JSON.stringify(keyboardEvent));

				e.preventDefault();
				return false;
			});
			window.addEventListener('keyup', function (e) {
				console.log('[client.js] 键盘释放事件:', {
					key: e.key,
					code: e.code,
					keyCode: e.keyCode,
					socketReady: self.socket && self.socket.readyState === WebSocket.OPEN,
					activeSession: self.activeSession,
					scancode: Mstsc.scancode(e)
				});
				
				// 检查并尝试修复连接状态
				if (!self.checkAndFixConnection()) {
					console.warn('[client.js] 无法修复连接状态，键盘事件被忽略');
					return;
				}
				
				if (!self.socket || self.socket.readyState !== WebSocket.OPEN || !self.activeSession) {
					console.warn('[client.js] 键盘事件被忽略:', {
						hasSocket: !!self.socket,
						socketState: self.socket ? self.socket.readyState : 'no socket',
						activeSession: self.activeSession
					});
					
					// 如果activeSession为false，尝试强制激活（用于调试）
					if (!self.activeSession && self.socket && self.socket.readyState === WebSocket.OPEN) {
						console.log('[client.js] 尝试强制激活会话');
						self.forceActivateSession();
					}
					return;
				}
				
				var scancode = Mstsc.scancode(e);
				if (!scancode || scancode === 0) {
					console.warn('[client.js] 无效的扫描码:', scancode, 'for key:', e.key, 'code:', e.code);
					return;
				}
				
				var keyboardEvent = {
					event: 'scancode',
					data: [scancode, false]
				};
				console.log('[client.js] 发送键盘释放事件:', keyboardEvent);
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
			
			console.log('[mstsc.js] 开始连接RDP:', {
				ip: ip,
				domain: domain,
				username: username,
				hasPassword: !!password,
				windowWs: typeof window.ws,
				windowWsState: window.ws ? window.ws.readyState : 'undefined'
			});
			
			// 检查是否已经有全局的WebSocket连接
			if (typeof window.ws !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
				console.log('[mstsc.js] 使用现有的WebSocket连接');
				this.socket = window.ws;
				
				// 设置消息处理器
				this.setupMessageHandler(next);
				
				// 发送连接信息
				this.sendConnectionInfo(ip, domain, username, password);
			} else {
				console.log('[mstsc.js] 没有可用的WebSocket连接，等待连接建立...');
				
				// 等待WebSocket连接建立
				var checkConnection = function() {
					console.log('[mstsc.js] 检查WebSocket连接状态:', {
						windowWs: typeof window.ws,
						windowWsState: window.ws ? window.ws.readyState : 'undefined',
						windowWsOpen: window.ws ? window.ws.readyState === WebSocket.OPEN : false
					});
					
					if (typeof window.ws !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
						console.log('[mstsc.js] WebSocket连接已建立，开始RDP连接');
						self.socket = window.ws;
						
						// 设置消息处理器
						self.setupMessageHandler(next);
						
						// 发送连接信息
						self.sendConnectionInfo(ip, domain, username, password);
					} else {
						console.log('[mstsc.js] 等待WebSocket连接...');
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
			
			console.log('[mstsc.js] 设置消息处理器:', {
				hasSocket: !!this.socket,
				socketState: this.socket ? this.socket.readyState : 'no socket',
				activeSession: this.activeSession
			});
			
			// 保存原有的消息处理器
			var originalOnMessage = this.socket.onmessage;
			
			this.socket.onmessage = function(event) {
				try {
					var message = JSON.parse(event.data);
					console.log('[mstsc.js] 收到消息:', message.event);
					
					switch(message.event) {
						case 'rdp-connect':
							console.log('[mstsc.js] RDP连接成功');
							if (message.data && message.data.reused) {
								console.log('[mstsc.js] 复用现有RDP连接');
								// 可以在这里添加复用连接的提示信息
								if (typeof window.showReusedConnectionMessage === 'function') {
									window.showReusedConnectionMessage(message.data.message || '复用现有RDP连接');
								}
							}
							self.activeSession = true;
							console.log('[mstsc.js] 设置activeSession为true，现在可以处理键盘事件');
							break;
						case 'rdp-bitmap':
							console.log('=== [client.js] 接收到bitmap更新事件 ===');
							console.log('[client.js] bitmap更新详情:', {
								bitsPerPixel: message.data.bitsPerPixel,
								rectanglesCount: message.data.rectangles ? message.data.rectangles.length : 0,
								timestamp: message.data.timestamp,
								hasRectangles: !!message.data.rectangles,
								isArray: Array.isArray(message.data.rectangles)
							});
							
							// 处理多个矩形数据
							if (message.data.rectangles && Array.isArray(message.data.rectangles)) {
								console.log('[client.js] 开始处理', message.data.rectangles.length, '个矩形');
								
								message.data.rectangles.forEach(function(rect, index) {
									console.log('[client.js] 处理矩形', index + 1, '/', message.data.rectangles.length, ':', {
										destLeft: rect.destLeft,
										destTop: rect.destTop,
										destRight: rect.destRight,
										destBottom: rect.destBottom,
										width: rect.width,
										height: rect.height,
										bitsPerPixel: rect.bitsPerPixel,
										isCompress: rect.isCompress,
										dataLength: rect.data ? rect.data.length : 0,
										dataType: rect.data ? rect.data.constructor.name : 'undefined'
									});
									
									// 验证矩形数据
									if (!rect.data || rect.data.length === 0) {
										console.error('[client.js] 矩形', index, '没有数据');
										return;
									}
									
									if (rect.width <= 0 || rect.height <= 0) {
										console.error('[client.js] 矩形', index, '尺寸无效:', rect.width, 'x', rect.height);
										return;
									}
									
									if (rect.destRight < rect.destLeft || rect.destBottom < rect.destTop) {
										console.error('[client.js] 矩形', index, '坐标无效');
										return;
									}
									
									// 后端已经解压缩，前端直接渲染
									console.log('[client.js] 矩形', index, '后端已解压缩，直接渲染');
									
									// 处理Base64编码的数据
									var decodedData = null;
									if (typeof rect.data === 'string') {
										// 如果是Base64字符串，解码为Uint8Array
										console.log('[client.js] 矩形', index, '检测到Base64编码数据，开始解码');
										try {
											// 将Base64字符串转换为二进制数据
											var binaryString = atob(rect.data);
											decodedData = new Uint8Array(binaryString.length);
											for (var k = 0; k < binaryString.length; k++) {
												decodedData[k] = binaryString.charCodeAt(k);
											}
											console.log('[client.js] 矩形', index, 'Base64解码成功，解码后长度:', decodedData.length);
										} catch (e) {
											console.error('[client.js] 矩形', index, 'Base64解码失败:', e);
											return;
										}
									} else {
										// 如果不是字符串，直接使用原始数据
										decodedData = rect.data;
									}
									
									// 验证数据长度是否正确（RGBA格式，4字节/像素）
									var expectedDataLength = rect.width * rect.height * 4;
									if (decodedData.length !== expectedDataLength) {
										console.warn('[client.js] 矩形', index, '数据长度不匹配，期望:', expectedDataLength, '实际:', decodedData.length);
										
										// 添加详细的数据类型调试信息
										console.log('[client.js] 矩形', index, '数据类型详情:', {
											dataType: typeof decodedData,
											constructor: decodedData ? decodedData.constructor.name : 'undefined',
											isArray: Array.isArray(decodedData),
											isUint8Array: decodedData instanceof Uint8Array,
											isArrayBuffer: decodedData instanceof ArrayBuffer,
											hasSlice: decodedData && typeof decodedData.slice === 'function',
											hasLength: decodedData && typeof decodedData.length !== 'undefined'
										});
										
										// 如果数据长度不匹配，尝试调整
										if (decodedData.length < expectedDataLength) {
											console.error('[client.js] 矩形', index, '数据不足，跳过');
											return;
										}
										// 如果数据过多，截断到正确长度
										if (decodedData.length > expectedDataLength) {
											console.warn('[client.js] 矩形', index, '数据过多，截断到正确长度');
											// 修复：使用正确的方式截断数据，避免数据丢失
											var originalData = decodedData;
											var truncatedData = null;
											
											if (decodedData instanceof Uint8Array) {
												console.log('[client.js] 矩形', index, '处理Uint8Array类型数据');
												truncatedData = decodedData.slice(0, expectedDataLength);
											} else if (decodedData instanceof ArrayBuffer) {
												console.log('[client.js] 矩形', index, '处理ArrayBuffer类型数据');
												truncatedData = decodedData.slice(0, expectedDataLength);
											} else if (Array.isArray(decodedData)) {
												console.log('[client.js] 矩形', index, '处理Array类型数据');
												truncatedData = decodedData.slice(0, expectedDataLength);
											} else if (decodedData && typeof decodedData.slice === 'function') {
												console.log('[client.js] 矩形', index, '处理具有slice方法的数据');
												truncatedData = decodedData.slice(0, expectedDataLength);
											} else {
												// 如果是其他类型的数据，尝试转换为Uint8Array
												console.log('[client.js] 矩形', index, '尝试转换为Uint8Array');
												try {
													var tempArray = new Uint8Array(decodedData);
													truncatedData = tempArray.slice(0, expectedDataLength);
												} catch (e) {
													console.error('[client.js] 矩形', index, '无法处理数据类型:', typeof decodedData, '错误:', e);
													return;
												}
											}
											
											// 验证截断结果
											if (truncatedData && truncatedData.length === expectedDataLength) {
												console.log('[client.js] 矩形', index, '数据截断成功:', {
													originalLength: originalData.length,
													truncatedLength: truncatedData.length,
													expectedLength: expectedDataLength,
													truncatedType: truncatedData.constructor.name
												});
												decodedData = truncatedData;
											} else {
												console.error('[client.js] 矩形', index, '数据截断失败:', {
													truncatedData: !!truncatedData,
													truncatedLength: truncatedData ? truncatedData.length : 'undefined',
													expectedLength: expectedDataLength
												});
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
									
									console.log('[client.js] 准备调用canvas.render.update()处理矩形', index);
									try {
										self.render.update(processedRect);
										console.log('[client.js] 矩形', index, 'canvas渲染调用完成');
									} catch (e) {
										console.error('[client.js] 矩形', index, 'canvas渲染失败:', e);
										console.error('[client.js] 错误堆栈:', e.stack);
									}
								});
								
								console.log('[client.js] 所有矩形处理完成');
							} else {
								console.warn('[client.js] 没有矩形数据或格式无效');
								console.log('[client.js] message.data内容:', message.data);
								
								// 兼容单个位图对象的情况
								try {
									console.log('[client.js] 尝试处理单个bitmap对象');
									self.render.update(message.data);
									console.log('[client.js] 单个bitmap对象处理完成');
								} catch (e) {
									console.error('[client.js] 单个bitmap对象处理失败:', e);
									console.error('[client.js] 错误堆栈:', e.stack);
								}
							}
							
							console.log('=== [client.js] bitmap更新事件处理完成 ===');
							break;
						case 'rdp-close':
							next(null);
							console.log('[mstsc.js] RDP连接关闭');
							self.activeSession = false;
							break;
						case 'rdp-error':
							next(message.data);
							console.log('[mstsc.js] RDP连接错误:', message.data);
							self.activeSession = false;
							break;
					}
					
					// 调用原有的消息处理器（如果存在）
					if (originalOnMessage) {
						originalOnMessage.call(this, event);
					}
				} catch (e) {
					console.error('[mstsc.js] 解析消息失败:', e);
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
			
			console.log('[mstsc.js] 发送连接信息:', {
				ip: infos.data.ip,
				port: infos.data.port,
				screen: infos.data.screen,
				domain: infos.data.domain,
				username: infos.data.username,
				hasPassword: !!infos.data.password
			});
			
			this.socket.send(JSON.stringify(infos));
		},
		
		/**
		 * 处理位图消息（从 index.html 转发）
		 */
		handleBitmapMessage : function(message) {
			console.log('=== [client.js] 通过 handleBitmapMessage 接收到bitmap更新事件 ===');
			console.log('[client.js] bitmap更新详情:', {
				bitsPerPixel: message.data.bitsPerPixel,
				rectanglesCount: message.data.rectangles ? message.data.rectangles.length : 0,
				timestamp: message.data.timestamp,
				hasRectangles: !!message.data.rectangles,
				isArray: Array.isArray(message.data.rectangles)
			});
			
			// 处理多个矩形数据
			if (message.data.rectangles && Array.isArray(message.data.rectangles)) {
				console.log('[client.js] 开始处理', message.data.rectangles.length, '个矩形');
				
				message.data.rectangles.forEach(function(rect, index) {
					console.log('[client.js] 处理矩形', index + 1, '/', message.data.rectangles.length, ':', {
						destLeft: rect.destLeft,
						destTop: rect.destTop,
						destRight: rect.destRight,
						destBottom: rect.destBottom,
						width: rect.width,
						height: rect.height,
						bitsPerPixel: rect.bitsPerPixel,
						isCompress: rect.isCompress,
						dataLength: rect.data ? rect.data.length : 0,
						dataType: rect.data ? rect.data.constructor.name : 'undefined'
					});
					
					// 验证矩形数据
					if (!rect.data || rect.data.length === 0) {
						console.error('[client.js] 矩形', index, '没有数据');
						return;
					}
					
					if (rect.width <= 0 || rect.height <= 0) {
						console.error('[client.js] 矩形', index, '尺寸无效:', rect.width, 'x', rect.height);
						return;
					}
					
					if (rect.destRight < rect.destLeft || rect.destBottom < rect.destTop) {
						console.error('[client.js] 矩形', index, '坐标无效');
						return;
					}
					
					// 后端已经解压缩，前端直接渲染
					console.log('[client.js] 矩形', index, '后端已解压缩，直接渲染');
					
					// 处理Base64编码的数据
					var decodedData = null;
					if (typeof rect.data === 'string') {
						// 如果是Base64字符串，解码为Uint8Array
						console.log('[client.js] 矩形', index, '检测到Base64编码数据，开始解码');
						try {
							// 将Base64字符串转换为二进制数据
							var binaryString = atob(rect.data);
							decodedData = new Uint8Array(binaryString.length);
							for (var k = 0; k < binaryString.length; k++) {
								decodedData[k] = binaryString.charCodeAt(k);
							}
							console.log('[client.js] 矩形', index, 'Base64解码成功，解码后长度:', decodedData.length);
						} catch (e) {
							console.error('[client.js] 矩形', index, 'Base64解码失败:', e);
							return;
						}
					} else {
						// 如果不是字符串，直接使用原始数据
						decodedData = rect.data;
					}
					
					// 验证数据长度是否正确（RGBA格式，4字节/像素）
					var expectedDataLength = rect.width * rect.height * 4;
					if (decodedData.length !== expectedDataLength) {
						console.warn('[client.js] 矩形', index, '数据长度不匹配，期望:', expectedDataLength, '实际:', decodedData.length);
						
						// 添加详细的数据类型调试信息
						console.log('[client.js] 矩形', index, '数据类型详情:', {
							dataType: typeof decodedData,
							constructor: decodedData ? decodedData.constructor.name : 'undefined',
							isArray: Array.isArray(decodedData),
							isUint8Array: decodedData instanceof Uint8Array,
							isArrayBuffer: decodedData instanceof ArrayBuffer,
							hasSlice: decodedData && typeof decodedData.slice === 'function',
							hasLength: decodedData && typeof decodedData.length !== 'undefined'
						});
						
						// 如果数据长度不匹配，尝试调整
						if (decodedData.length < expectedDataLength) {
							console.error('[client.js] 矩形', index, '数据不足，跳过');
							return;
						}
						// 如果数据过多，截断到正确长度
						if (decodedData.length > expectedDataLength) {
							console.warn('[client.js] 矩形', index, '数据过多，截断到正确长度');
							// 修复：使用正确的方式截断数据，避免数据丢失
							var originalData = decodedData;
							var truncatedData = null;
							
							if (decodedData instanceof Uint8Array) {
								console.log('[client.js] 矩形', index, '处理Uint8Array类型数据');
								truncatedData = decodedData.slice(0, expectedDataLength);
							} else if (decodedData instanceof ArrayBuffer) {
								console.log('[client.js] 矩形', index, '处理ArrayBuffer类型数据');
								truncatedData = decodedData.slice(0, expectedDataLength);
							} else if (Array.isArray(decodedData)) {
								console.log('[client.js] 矩形', index, '处理Array类型数据');
								truncatedData = decodedData.slice(0, expectedDataLength);
							} else if (decodedData && typeof decodedData.slice === 'function') {
								console.log('[client.js] 矩形', index, '处理具有slice方法的数据');
								truncatedData = decodedData.slice(0, expectedDataLength);
							} else {
								// 如果是其他类型的数据，尝试转换为Uint8Array
								console.log('[client.js] 矩形', index, '尝试转换为Uint8Array');
								try {
									var tempArray = new Uint8Array(decodedData);
									truncatedData = tempArray.slice(0, expectedDataLength);
								} catch (e) {
									console.error('[client.js] 矩形', index, '无法处理数据类型:', typeof decodedData, '错误:', e);
									return;
								}
							}
							
							// 验证截断结果
							if (truncatedData && truncatedData.length === expectedDataLength) {
								console.log('[client.js] 矩形', index, '数据截断成功:', {
									originalLength: originalData.length,
									truncatedLength: truncatedData.length,
									expectedLength: expectedDataLength,
									truncatedType: truncatedData.constructor.name
								});
								decodedData = truncatedData;
							} else {
								console.error('[client.js] 矩形', index, '数据截断失败:', {
									truncatedData: !!truncatedData,
									truncatedLength: truncatedData ? truncatedData.length : 'undefined',
									expectedLength: expectedDataLength
								});
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
					
					console.log('[client.js] 准备调用canvas.render.update()处理矩形', index);
					try {
						this.render.update(processedRect);
						console.log('[client.js] 矩形', index, 'canvas渲染调用完成');
					} catch (e) {
						console.error('[client.js] 矩形', index, 'canvas渲染失败:', e);
						console.error('[client.js] 错误堆栈:', e.stack);
					}
				}.bind(this));
				
				console.log('[client.js] 所有矩形处理完成');
			} else {
				console.warn('[client.js] 没有矩形数据或格式无效');
				console.log('[client.js] message.data内容:', message.data);
				
				// 兼容单个位图对象的情况
				try {
					console.log('[client.js] 尝试处理单个bitmap对象');
					this.render.update(message.data);
					console.log('[client.js] 单个bitmap对象处理完成');
				} catch (e) {
					console.error('[client.js] 单个bitmap对象处理失败:', e);
					console.error('[client.js] 错误堆栈:', e.stack);
				}
			}
			
			console.log('=== [client.js] handleBitmapMessage 处理完成 ===');
		},
		
		/**
		 * 检查连接状态并尝试修复
		 */
		checkAndFixConnection : function() {
			var self = this;
			
			console.log('[client.js] 检查连接状态:', {
				hasSocket: !!this.socket,
				socketState: this.socket ? this.socket.readyState : 'no socket',
				activeSession: this.activeSession,
				windowWs: typeof window.ws,
				windowWsState: window.ws ? window.ws.readyState : 'undefined'
			});
			
			// 如果socket不存在或已关闭，尝试重新获取
			if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
				console.log('[client.js] Socket状态异常，尝试重新获取');
				if (typeof window.ws !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
					console.log('[client.js] 找到可用的WebSocket连接，重新设置');
					this.socket = window.ws;
					return true;
				} else {
					console.warn('[client.js] 没有可用的WebSocket连接');
					return false;
				}
			}
			
			return true;
		},
		
		/**
		 * 强制激活会话（用于调试）
		 */
		forceActivateSession : function() {
			console.log('[client.js] 强制激活会话');
			this.activeSession = true;
			console.log('[client.js] activeSession已设置为true');
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
