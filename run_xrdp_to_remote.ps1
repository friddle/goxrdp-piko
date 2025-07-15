# Windows PowerShell脚本 - 下载goxrdp并连接远程终端
# 作者: 基于run_local.sh配置生成

# 设置错误处理
$ErrorActionPreference = "Stop"

Write-Host "=== goxrdp Windows 远程连接工具 ===" -ForegroundColor Green

# 获取当前IP地址
$currentIP = (Get-NetIPAddress -AddressFamily IPv4 | Where-Object {$_.IPAddress -notlike "127.*" -and $_.IPAddress -notlike "169.*"} | Select-Object -First 1).IPAddress
Write-Host "当前IP地址: $currentIP" -ForegroundColor Yellow

# 询问用户是否控制本机
Write-Host ""
$isLocalControl = Read-Host "您是要控制本机吗？(y/n)"
if ($isLocalControl -eq "y" -or $isLocalControl -eq "Y") {
    Write-Host ""
    Write-Host "??  警告：您选择了控制本机！" -ForegroundColor Red
    Write-Host "   这将中断当前的远程连接和所有操作" -ForegroundColor Red
    Write-Host "   请确保已保存所有工作" -ForegroundColor Yellow
    Write-Host ""
    $confirmLocal = Read-Host "确认要继续吗？(y/n)"
    if ($confirmLocal -ne "y" -and $confirmLocal -ne "Y") {
        Write-Host "操作已取消" -ForegroundColor Yellow
        exit 0
    }
    $xrdpHost = $currentIP
    Write-Host "使用本机IP: $xrdpHost" -ForegroundColor Green
} else {
    $xrdpHost = Read-Host "请输入要控制的远程主机IP地址"
    Write-Host "使用远程IP: $xrdpHost" -ForegroundColor Green
}

# 询问用户输入
Write-Host ""
$username = Read-Host "请输入用户名"
$password = Read-Host "请输入密码" -AsSecureString
$passwordPlain = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($password))

# 询问连接名称，默认为用户名
Write-Host ""
$defaultConnectionName = $username
$connectionName = Read-Host "请输入连接名称 (默认: $defaultConnectionName)"
if ([string]::IsNullOrWhiteSpace($connectionName)) {
    $connectionName = $defaultConnectionName
}

# 远程服务器配置
$remoteServer = "http://piko-upstream.inner-service.laiye.com:8089"

Write-Host ""

# 检查当前目录是否已有goxrdp.exe
$currentDirPath = ".\goxrdp.exe"
if (Test-Path $currentDirPath) {
    Write-Host "发现当前目录下已有goxrdp.exe，直接使用..." -ForegroundColor Green
    $localPath = $currentDirPath
} else {
    Write-Host "当前目录下没有goxrdp.exe，正在下载Windows版本的goxrdp..." -ForegroundColor Cyan

    # 创建临时目录
    $tempDir = "$env:TEMP\goxrdp"
    if (-not (Test-Path $tempDir)) {
        New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    }

    # 下载goxrdp Windows版本
    # 注意：这里需要根据实际的下载地址进行调整
    $downloadUrl = "https://oss-inner.laiye.com/laiye-devops/third/goxrdp-windows-amd64.exe"
    $localPath = ".\goxrdp.exe"

    try {
        Write-Host "从 $downloadUrl 下载..." -ForegroundColor Yellow
        Invoke-WebRequest -Uri $downloadUrl -OutFile $localPath -UseBasicParsing
        Write-Host "下载完成！" -ForegroundColor Green
    }
    catch {
        Write-Host "下载失败，请检查网络连接或下载地址是否正确" -ForegroundColor Red
        Write-Host "错误信息: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }

    # 验证文件是否存在
    if (-not (Test-Path $localPath)) {
        Write-Host "下载的文件不存在，请检查下载地址" -ForegroundColor Red
        exit 1
    }
}

Write-Host "访问地址为: http://192.168.10.95:8080/$connectionName/html/" -ForegroundColor Yellow
Write-Host ""
Write-Host "正在启动远程连接..." -ForegroundColor Cyan
Write-Host "连接信息:" -ForegroundColor Yellow
Write-Host "  名称: $connectionName" -ForegroundColor White
Write-Host "  远程服务器: $remoteServer" -ForegroundColor White
Write-Host "  XRDP主机: $xrdpHost" -ForegroundColor White
Write-Host "  用户名: $username" -ForegroundColor White
Write-Host ""

# 启动goxrdp
try {
    & $localPath `
        "--name=$connectionName" `
        "--remote=$remoteServer" `
        "--xrdp-host=$xrdpHost" `
        "--xrdp-user=$username" `
        "--xrdp-pass=$passwordPlain"
}
catch {
    Write-Host "启动goxrdp失败: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "连接已关闭" -ForegroundColor Green

