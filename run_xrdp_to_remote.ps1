# Windows PowerShell�ű� - ����goxrdp������Զ���ն�
# ����: ����run_local.sh��������

# ���ô�����
$ErrorActionPreference = "Stop"

Write-Host "=== goxrdp Windows Զ�����ӹ��� ===" -ForegroundColor Green

# ��ȡ��ǰIP��ַ
$currentIP = (Get-NetIPAddress -AddressFamily IPv4 | Where-Object {$_.IPAddress -notlike "127.*" -and $_.IPAddress -notlike "169.*"} | Select-Object -First 1).IPAddress
Write-Host "��ǰIP��ַ: $currentIP" -ForegroundColor Yellow

# ѯ���û��Ƿ���Ʊ���
Write-Host ""
$isLocalControl = Read-Host "����Ҫ���Ʊ�����(y/n)"
if ($isLocalControl -eq "y" -or $isLocalControl -eq "Y") {
    Write-Host ""
    Write-Host "??  ���棺��ѡ���˿��Ʊ�����" -ForegroundColor Red
    Write-Host "   �⽫�жϵ�ǰ��Զ�����Ӻ����в���" -ForegroundColor Red
    Write-Host "   ��ȷ���ѱ������й���" -ForegroundColor Yellow
    Write-Host ""
    $confirmLocal = Read-Host "ȷ��Ҫ������(y/n)"
    if ($confirmLocal -ne "y" -and $confirmLocal -ne "Y") {
        Write-Host "������ȡ��" -ForegroundColor Yellow
        exit 0
    }
    $xrdpHost = $currentIP
    Write-Host "ʹ�ñ���IP: $xrdpHost" -ForegroundColor Green
} else {
    $xrdpHost = Read-Host "������Ҫ���Ƶ�Զ������IP��ַ"
    Write-Host "ʹ��Զ��IP: $xrdpHost" -ForegroundColor Green
}

# ѯ���û�����
Write-Host ""
$username = Read-Host "�������û���"
$password = Read-Host "����������" -AsSecureString
$passwordPlain = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($password))

# ѯ���������ƣ�Ĭ��Ϊ�û���
Write-Host ""
$defaultConnectionName = $username
$connectionName = Read-Host "�������������� (Ĭ��: $defaultConnectionName)"
if ([string]::IsNullOrWhiteSpace($connectionName)) {
    $connectionName = $defaultConnectionName
}

# Զ�̷���������
$remoteServer = "http://piko-upstream.inner-service.laiye.com:8089"

Write-Host ""

# ��鵱ǰĿ¼�Ƿ�����goxrdp.exe
$currentDirPath = ".\goxrdp.exe"
if (Test-Path $currentDirPath) {
    Write-Host "���ֵ�ǰĿ¼������goxrdp.exe��ֱ��ʹ��..." -ForegroundColor Green
    $localPath = $currentDirPath
} else {
    Write-Host "��ǰĿ¼��û��goxrdp.exe����������Windows�汾��goxrdp..." -ForegroundColor Cyan

    # ������ʱĿ¼
    $tempDir = "$env:TEMP\goxrdp"
    if (-not (Test-Path $tempDir)) {
        New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    }

    # ����goxrdp Windows�汾
    # ע�⣺������Ҫ����ʵ�ʵ����ص�ַ���е���
    $downloadUrl = "https://oss-inner.laiye.com/laiye-devops/third/goxrdp-windows-amd64.exe"
    $localPath = ".\goxrdp.exe"

    try {
        Write-Host "�� $downloadUrl ����..." -ForegroundColor Yellow
        Invoke-WebRequest -Uri $downloadUrl -OutFile $localPath -UseBasicParsing
        Write-Host "������ɣ�" -ForegroundColor Green
    }
    catch {
        Write-Host "����ʧ�ܣ������������ӻ����ص�ַ�Ƿ���ȷ" -ForegroundColor Red
        Write-Host "������Ϣ: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }

    # ��֤�ļ��Ƿ����
    if (-not (Test-Path $localPath)) {
        Write-Host "���ص��ļ������ڣ��������ص�ַ" -ForegroundColor Red
        exit 1
    }
}

Write-Host "���ʵ�ַΪ: http://192.168.10.95:8080/$connectionName/html/" -ForegroundColor Yellow
Write-Host ""
Write-Host "��������Զ������..." -ForegroundColor Cyan
Write-Host "������Ϣ:" -ForegroundColor Yellow
Write-Host "  ����: $connectionName" -ForegroundColor White
Write-Host "  Զ�̷�����: $remoteServer" -ForegroundColor White
Write-Host "  XRDP����: $xrdpHost" -ForegroundColor White
Write-Host "  �û���: $username" -ForegroundColor White
Write-Host ""

# ����goxrdp
try {
    & $localPath `
        "--name=$connectionName" `
        "--remote=$remoteServer" `
        "--xrdp-host=$xrdpHost" `
        "--xrdp-user=$username" `
        "--xrdp-pass=$passwordPlain"
}
catch {
    Write-Host "����goxrdpʧ��: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "�����ѹر�" -ForegroundColor Green

