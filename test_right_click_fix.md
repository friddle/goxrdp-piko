# 鼠标右键修复测试指南

## 问题描述

**原始问题：** 鼠标右键被错误地识别为拖拽操作，导致右键点击无法正常工作。

## 修复内容

### 1. 问题分析

**问题根源：**
- 在鼠标拖拽逻辑中，任何按钮被按下时都会设置 `isDragging = true`
- 鼠标移动时，会发送带有按下状态的鼠标事件
- 这导致右键被识别为拖拽而不是右键点击

**修复方案：**
- 添加了 `isDragOperation()` 函数来区分拖拽操作
- 只有左键（button=0）才允许进行拖拽操作
- 右键（button=2）和中键（button=1）不进行拖拽

### 2. 具体修改

#### 新增函数
```javascript
function isDragOperation(button, pressedButtons) {
    // 只有左键（button=0）才允许拖拽操作
    // 右键（button=2）和中键（button=1）不进行拖拽
    return button === 0 && pressedButtons.has(0);
}
```

#### 修改鼠标移动逻辑
```javascript
// 修复前：任何按钮都可以拖拽
if (self.mouseState.isDragging && self.mouseState.pressedButtons.size > 0) {
    var button = Array.from(self.mouseState.pressedButtons)[0];
    // 发送拖拽事件
}

// 修复后：只有左键才拖拽
if (self.mouseState.isDragging && isDragOperation(0, self.mouseState.pressedButtons)) {
    // 只有左键拖拽时才发送拖拽事件
}
```

#### 修改拖拽状态管理
```javascript
// 修复前：任何按钮按下都设置拖拽状态
self.mouseState.isDragging = true;

// 修复后：只有左键按下才设置拖拽状态
if (mappedButton === 0) {
    self.mouseState.isDragging = true;
}
```

## 测试方法

### 1. 启动应用
```bash
cd /home/friddle/project/friddle_projects/grdp
./dist/goxrdp
```

### 2. 连接RDP
- 打开浏览器访问应用
- 输入RDP连接信息并连接

### 3. 测试右键功能

#### 测试右键菜单
1. 在RDP会话中，右键点击桌面空白区域
2. 观察是否出现右键菜单
3. 右键点击文件或文件夹
4. 观察是否出现相应的右键菜单

#### 测试右键拖拽（应该被禁用）
1. 在RDP会话中，按住右键并移动鼠标
2. 观察是否不会触发拖拽操作
3. 确认右键只用于显示菜单，不用于拖拽

#### 测试左键拖拽（应该正常工作）
1. 在RDP会话中，按住左键并移动鼠标
2. 观察拖拽操作是否正常
3. 尝试拖拽窗口标题栏
4. 尝试拖拽文件或文件夹

### 4. 观察日志

#### 后端日志
```bash
# 查看鼠标事件日志
grep "鼠标事件" logs/app.log
grep "转发鼠标" logs/app.log
```

#### 前端日志
在浏览器开发者工具的控制台中观察：
- 鼠标事件是否正确发送
- 右键事件是否被正确识别

### 5. 预期结果

**正常情况：**
- ✅ 右键点击显示菜单
- ✅ 右键不会触发拖拽
- ✅ 左键拖拽正常工作
- ✅ 鼠标事件在日志中正确记录

**异常情况：**
- ❌ 右键仍然被识别为拖拽
- ❌ 右键菜单不显示
- ❌ 左键拖拽失效

## 调试信息

### 鼠标状态跟踪
修复后的代码会正确跟踪以下状态：
- `isDragging`: 只有左键按下时为true
- `pressedButtons`: 跟踪所有按下的按钮
- 右键按下时不会设置拖拽状态

### 事件处理逻辑
- 右键按下：发送右键按下事件，不设置拖拽状态
- 右键移动：发送普通移动事件，不发送拖拽事件
- 右键释放：发送右键释放事件
- 左键按下：发送左键按下事件，设置拖拽状态
- 左键移动：如果正在拖拽，发送左键拖拽事件
- 左键释放：发送左键释放事件，清除拖拽状态

## 相关文件

- `client_piko/web/js/client.js` - 主要修复文件
- `client_piko/web.go` - 后端鼠标事件处理
- `client_piko/rdp_client.go` - RDP客户端鼠标事件处理

## 注意事项

1. 确保RDP连接稳定
2. 检查网络延迟是否影响右键响应
3. 如果仍有问题，可以启用调试模式：
   ```bash
   export IMAGE_DEBUG=true
   ./dist/goxrdp
   ```

## 验证步骤

1. **右键菜单测试**
   - 右键点击桌面 → 应该显示桌面右键菜单
   - 右键点击文件 → 应该显示文件右键菜单
   - 右键点击文件夹 → 应该显示文件夹右键菜单

2. **拖拽功能测试**
   - 左键拖拽窗口 → 应该可以移动窗口
   - 左键拖拽文件 → 应该可以移动文件
   - 右键拖拽 → 不应该触发拖拽操作

3. **日志验证**
   - 右键事件应该显示正确的按钮编号（2）
   - 左键拖拽事件应该显示正确的按钮编号（0）
   - 不应该有右键拖拽的日志记录 