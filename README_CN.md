# grdp-piko

一个基于grdp协议和piko网络转发的Windows RDP远程桌面工具。通过Web浏览器访问和控制Windows远程桌面，无需复杂的网络配置和外网地址。

[grdp](https://github.com/friddle/grdp) - Go语言实现的RDP协议客户端
[piko](https://github.com/andydunstall/piko) - 轻量级网络隧道服务

## 项目特点

- 🖥️ **Web远程桌面**: 通过浏览器访问Windows远程桌面
- 🚀 **轻量级**: 基于Go语言实现，资源占用低
- 🔧 **简单部署**: 后端Docker一键部署，配置简单
- 🔒 **安全可靠**: 基于RDP协议，支持用户认证
- 📱 **跨平台**: 支持Linux、macOS,windows客户端
- 🌐 **网络穿透**: 通过piko服务实现内网穿透

## 架构说明

```
Windows RDP服务器 (3389端口)
    ↓ RDP协议
grdp客户端 (goxrdp)
    ↓ piko隧道
piko服务器
    ↓ HTTP/WebSocket
Web浏览器
```

## 快速开始

### 服务端部署

1. **使用 Docker Compose 部署**

```yaml
# docker-compose.yaml
version: "3.8"
services:
  piko:
    image: ghcr.io/friddle/grdp-piko-server:latest
    container_name: grdp-piko-server
    environment:
      - PIKO_UPSTREAM_PORT=8022
      - LISTEN_PORT=8088
    ports:
      - "8022:8022"
      - "8088:8088"
    restart: unless-stopped
```

或直接使用 Docker：

```bash
docker run -ti --network=host --rm --name=piko-server ghcr.io/friddle/grdp-piko-server
```

2. **启动服务**

```bash
docker-compose up -d
```

### 客户端使用

#### Linux 客户端

```bash
# 下载客户端
wget https://github.com/friddle/grdp/releases/download/v1.0.0/goxrdp-linux-amd64 -O ./goxrdp
chmod +x ./goxrdp

# 基本连接
./goxrdp --name=windows-server --remote=192.168.1.100:8088

# 指定RDP服务器和用户
./goxrdp --name=windows-server --remote=192.168.1.100:8088 \
  --xrdp-host=192.168.1.200 \
  --xrdp-user=Administrator \
  --xrdp-pass=password

# 禁用自动退出
./goxrdp --name=windows-server --remote=192.168.1.100:8088 --auto-exit=false
```

#### macOS 客户端

```bash
# 下载客户端
curl -L -o goxrdp https://github.com/friddle/grdp/releases/download/v1.0.1/goxrdp-darwin-amd64
chmod +x ./goxrdp

# 连接Windows RDP服务器
./goxrdp --name=windows-server --remote=192.168.1.100:8088 \
  --xrdp-host=192.168.1.200 \
  --xrdp-user=Administrator
```

![客户端启动截图](screenshot/start_cli.png)
![Web远程桌面截图](screenshot/webui.png)

## 访问方式

当客户端启动后，通过以下地址访问Windows远程桌面：
```
http://主机服务器IP:端口/客户端名称
```

例如：
- 服务端监听的地址: `192.168.1.100:8088`
- 客户端名称: `windows-server`
- 访问地址: `http://192.168.1.100:8088/windows-server`

## 配置说明

### 客户端参数

| 参数 | 说明 | 默认值 | 必填 |
|------|------|--------|------|
| `--name` | piko 客户端标识名称 | - | ✅ |
| `--remote` | 远程 piko 服务器地址 (格式: host:port) | - | ✅ |
| `--xrdp-host` | Windows RDP服务器主机地址 | 自动获取本机IP | ❌ |
| `--xrdp-port` | Windows RDP服务器端口 | 3389 | ❌ |
| `--xrdp-user` | Windows RDP用户名 | 自动获取当前用户 | ❌ |
| `--xrdp-pass` | Windows RDP密码 | - | ❌ |
| `--xrdp-domain` | Windows RDP域 (为空时使用本地计算机名) | 自动获取计算机名 | ❌ |
| `--auto-exit` | 是否启用24小时自动退出 | true | ❌ |

### 服务端环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PIKO_UPSTREAM_PORT` | Piko 上游端口 | 8022 |
| `LISTEN_PORT` | HTTP 监听端口 | 8088 |

### 使用场景

1. **内网穿透**: 通过piko服务将内网Windows服务器暴露到外网
2. **远程协助**: 通过Web浏览器远程控制Windows桌面
3. **开发调试**: 在远程Windows环境中进行开发和调试
4. **服务器管理**: 通过Web界面管理Windows服务器

### 安全注意事项

- 确保RDP服务器启用了网络级别身份验证(NLA)
- 使用强密码保护RDP账户
- 考虑使用VPN或防火墙限制访问
- 定期更新Windows系统和RDP服务

### 故障排除

1. **连接失败**: 检查Windows RDP服务是否启用(3389端口)
2. **认证失败**: 确认用户名和密码正确，检查账户权限
3. **网络问题**: 验证piko服务器地址和端口是否正确
4. **权限问题**: 确保客户端有足够的网络访问权限

## 技术栈

- **后端**: Go语言
- **RDP协议**: grdp (Go RDP客户端)
- **网络隧道**: piko
- **Web界面**: HTML5 Canvas + WebSocket
- **部署**: Docker

## 许可证

本项目基于MIT许可证开源。

