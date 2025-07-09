package client_piko

import (
	"fmt"
	"net"
	"time"
)

type RDPDiagnostic struct {
	host   string
	port   int
	logger func(format string, args ...any)
}

func NewRDPDiagnostic(host string, port int, logger func(format string, args ...any)) *RDPDiagnostic {
	return &RDPDiagnostic{
		host:   host,
		port:   port,
		logger: logger,
	}
}

func (d *RDPDiagnostic) RunDiagnostics() error {
	d.logger("🔍 开始RDP连接诊断...\n")

	// 1. 检查网络连通性
	if err := d.checkNetworkConnectivity(); err != nil {
		d.logger("❌ 网络连通性检查失败: %v\n", err)
		return err
	}

	// 2. 检查端口连通性
	if err := d.checkPortConnectivity(); err != nil {
		d.logger("❌ 端口连通性检查失败: %v\n", err)
		return err
	}

	// 3. 检查RDP服务状态
	if err := d.checkRDPService(); err != nil {
		d.logger("❌ RDP服务检查失败: %v\n", err)
		return err
	}

	d.logger("✅ RDP诊断完成\n")
	return nil
}

func (d *RDPDiagnostic) checkNetworkConnectivity() error {
	d.logger("   检查网络连通性: ping %s\n", d.host)

	// 简单的网络连通性检查
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", d.host, d.port), 5*time.Second)
	if err != nil {
		return fmt.Errorf("无法连接到 %s:%d: %v", d.host, d.port, err)
	}
	defer conn.Close()

	d.logger("   ✅ 网络连通性正常\n")
	return nil
}

func (d *RDPDiagnostic) checkPortConnectivity() error {
	d.logger("   检查端口连通性: %s:%d\n", d.host, d.port)

	// 检查端口是否开放
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", d.host, d.port), 3*time.Second)
	if err != nil {
		return fmt.Errorf("端口 %d 不可达: %v", d.port, err)
	}
	defer conn.Close()

	d.logger("   ✅ 端口连通性正常\n")
	return nil
}

func (d *RDPDiagnostic) checkRDPService() error {
	d.logger("   检查RDP服务状态\n")

	// 尝试建立RDP连接并检查响应
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", d.host, d.port), 5*time.Second)
	if err != nil {
		return fmt.Errorf("RDP服务不可达: %v", err)
	}
	defer conn.Close()

	// 发送简单的RDP连接请求
	rdpRequest := []byte{
		0x03, 0x00, // TPKT version 3
		0x00, 0x2c, // Length
		0x02, 0xf0, 0x80, // X.224 Connection Request
		0x7f, 0x65, 0x82, 0x01, 0x94, 0x04, 0x01, 0x01, 0x04, 0x01, 0x01, 0x01, 0x01, 0xff,
		0x30, 0x19, 0x02, 0x01, 0x22, 0x02, 0x01, 0x02, 0x02, 0x01, 0x00, 0x02, 0x01, 0x01,
		0x02, 0x01, 0x00, 0x02, 0x01, 0x01, 0x02, 0x02, 0xff, 0xff, 0x02, 0x01, 0x02,
	}

	_, err = conn.Write(rdpRequest)
	if err != nil {
		return fmt.Errorf("发送RDP请求失败: %v", err)
	}

	// 读取响应
	response := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		return fmt.Errorf("读取RDP响应失败: %v", err)
	}

	if n > 0 {
		d.logger("   ✅ RDP服务响应正常\n")
		return nil
	}

	return fmt.Errorf("RDP服务无响应")
}

// TestColorConversion 测试颜色转换函数
func TestColorConversion() {
	fmt.Println("=== 测试颜色转换函数 ===")

	// 测试16位色深转换
	fmt.Println("测试16位色深转换...")
	src16 := []byte{0xF8, 0x1F} // RGB565: 红色 (1111100000011111)
	dst16 := make([]byte, 4)
	convert16ToRGBA(src16, dst16, 1, 1)
	fmt.Printf("16位输入: [0x%02x, 0x%02x] -> RGBA输出: [%d, %d, %d, %d]\n",
		src16[0], src16[1], dst16[0], dst16[1], dst16[2], dst16[3])

	// 测试24位色深转换
	fmt.Println("测试24位色深转换...")
	src24 := []byte{0x00, 0x00, 0xFF} // BGR: 红色 (B=0, G=0, R=255)
	dst24 := make([]byte, 4)
	convert24ToRGBA(src24, dst24, 1, 1)
	fmt.Printf("24位输入: [0x%02x, 0x%02x, 0x%02x] -> RGBA输出: [%d, %d, %d, %d]\n",
		src24[0], src24[1], src24[2], dst24[0], dst24[1], dst24[2], dst24[3])

	// 测试32位色深转换
	fmt.Println("测试32位色深转换...")
	src32 := []byte{0x00, 0x00, 0xFF, 0xFF} // BGRA: 红色 (B=0, G=0, R=255, A=255)
	dst32 := make([]byte, 4)
	convert32ToRGBA(src32, dst32, 1, 1)
	fmt.Printf("32位输入: [0x%02x, 0x%02x, 0x%02x, 0x%02x] -> RGBA输出: [%d, %d, %d, %d]\n",
		src32[0], src32[1], src32[2], src32[3], dst32[0], dst32[1], dst32[2], dst32[3])

	fmt.Println("=== 颜色转换测试完成 ===")
}
