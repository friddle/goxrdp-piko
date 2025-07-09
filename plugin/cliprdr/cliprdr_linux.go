//go:build linux
// +build linux

package cliprdr

import (
	"github.com/friddle/grdp/glog"
)

// Linux版本的常量定义
const (
	FILE_ATTRIBUTE_DIRECTORY = 0x00000010
	CF_HDROP                 = 15
	CFSTR_FILEDESCRIPTORW    = "FileGroupDescriptorW"
	CFSTR_FILECONTENTS       = "FileContents"
)

// Linux版本的Control结构体
type Control struct {
	hwnd uintptr // Linux版本暂时不使用，但需要保持结构体兼容
}

// Linux版本的ClipWatcher函数
func ClipWatcher(c *CliprdrClient) {
	glog.Info("ClipWatcher: Linux版本暂不支持剪贴板监控")
	// Linux版本暂时不实现剪贴板监控
}

// Linux版本的withOpenClipboard方法
func (c *Control) withOpenClipboard(f func()) {
	glog.Info("withOpenClipboard: Linux版本暂不支持剪贴板操作")
	// Linux版本暂时不实现
}

// Linux版本的EmptyClipboard函数
func EmptyClipboard() bool {
	glog.Info("EmptyClipboard: Linux版本暂不支持剪贴板操作")
	return false
}

// Linux版本的SetClipboardData函数
func SetClipboardData(formatId uint32, hmem uintptr) bool {
	glog.Info("SetClipboardData: Linux版本暂不支持剪贴板操作")
	return false
}

// Linux版本的SendCliprdrMessage方法
func (c *Control) SendCliprdrMessage() {
	glog.Info("SendCliprdrMessage: Linux版本暂不支持剪贴板消息")
}

// Linux版本的GetFileInfo函数
func GetFileInfo(sys interface{}) (uint32, []byte, uint32, uint32) {
	glog.Info("GetFileInfo: Linux版本暂不支持文件信息获取")
	return 0, nil, 0, 0
}

// Linux版本的RegisterClipboardFormat函数
func RegisterClipboardFormat(format string) uint32 {
	glog.Info("RegisterClipboardFormat: Linux版本暂不支持剪贴板格式注册")
	return 0
}

// Linux版本的GetClipboardData函数
func GetClipboardData(formatId uint32) string {
	glog.Info("GetClipboardData: Linux版本暂不支持剪贴板数据获取")
	return ""
}

// Linux版本的GetFileNames函数
func GetFileNames() []string {
	glog.Info("GetFileNames: Linux版本暂不支持文件列表获取")
	return []string{}
}

// Linux版本的IsClipboardFormatAvailable函数
func IsClipboardFormatAvailable(id uint32) bool {
	glog.Info("IsClipboardFormatAvailable: Linux版本暂不支持剪贴板格式检查")
	return false
}

// Linux版本的GetFormatList函数
func GetFormatList(hwnd uintptr) []CliprdrFormat {
	glog.Info("GetFormatList: Linux版本暂不支持剪贴板格式列表获取")
	return []CliprdrFormat{}
}
