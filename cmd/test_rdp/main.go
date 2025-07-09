package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/friddle/grdp/client_piko"
)

func main() {
	// 定义命令行参数
	var (
		host     = flag.String("host", "192.168.16.36", "目标主机IP地址")
		user     = flag.String("user", "administrator", "用户名")
		password = flag.String("password", "password", "密码")
		domain   = flag.String("domain", "", "域名（可选）")
		port     = flag.Int("port", 3389, "RDP端口")
	)
	flag.Parse()

	// 验证必需参数
	if *host == "" || *user == "" || *password == "" {
		fmt.Println("使用方法:")
		flag.PrintDefaults()
		fmt.Println("\n示例:")
		fmt.Println("  go run cmd/test_rdp/main.go -host=192.168.16.36 -user=administrator -password=mypassword")
		fmt.Println("  go run cmd/test_rdp/main.go -host=192.168.16.36 -user=administrator -password=mypassword -domain=mydomain")
		os.Exit(1)
	}

	fmt.Printf("开始测试RDP连接到 %s:%d\n", *host, *port)
	fmt.Printf("用户: %s\n", *user)
	if *domain != "" {
		fmt.Printf("域名: %s\n", *domain)
	}
	fmt.Println("----------------------------------------")

	// 测试连接
	err := client_piko.TestConnection(*host, *user, *password, *domain)
	if err != nil {
		log.Fatalf("RDP连接测试失败: %v", err)
	}

	fmt.Println("----------------------------------------")
	fmt.Println("RDP连接测试成功完成！")
}
