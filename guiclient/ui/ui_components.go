package ui

import (
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/friddle/grdp/guiclient/config"
)

// UIState UI状态
type UIState struct {
	// 界面元素
	RemoteServerEntry   widget.Editor
	ConnectionNameEntry widget.Editor
	HostEntry           widget.Editor
	UsernameEntry       widget.Editor
	PasswordEntry       widget.Editor
	LocalControlCheck   widget.Bool
	ConnectBtn          widget.Clickable
	QuitBtn             widget.Clickable
	SaveConfigBtn       widget.Clickable

	// 状态信息
	StatusText   string
	StatusColor  string
	AccessURL    string
	IsConnected  bool
	LastError    string
	LastSaveTime int64 // 用于防抖

	// 配置
	Config *config.ConnectionConfig
}

// NewUIState 创建新的UI状态
func NewUIState() *UIState {
	state := &UIState{
		Config: &config.ConnectionConfig{},
	}

	// 设置编辑器属性
	state.RemoteServerEntry.SingleLine = true
	state.ConnectionNameEntry.SingleLine = true
	state.HostEntry.SingleLine = true
	state.UsernameEntry.SingleLine = true
	state.PasswordEntry.SingleLine = true
	state.PasswordEntry.Mask = '*'

	return state
}

// UpdateConnectionName 更新连接名称（与用户名同步）
func (state *UIState) UpdateConnectionName() {
	username := state.UsernameEntry.Text()
	if username != "" && state.ConnectionNameEntry.Text() == "" {
		state.ConnectionNameEntry.SetText(username)
		state.Config.ConnectionName = username
	}
}

// GetStatusColor 获取状态颜色
func (state *UIState) GetStatusColor() color.NRGBA {
	switch state.StatusColor {
	case "red":
		return color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	case "green":
		return color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	case "blue":
		return color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	default:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	}
}

// ShouldAutoSave 检查是否应该自动保存
func (state *UIState) ShouldAutoSave() bool {
	currentTime := time.Now().Unix()
	return state.RemoteServerEntry.Text() != "" &&
		state.HostEntry.Text() != "" &&
		state.UsernameEntry.Text() != "" &&
		state.PasswordEntry.Text() != "" &&
		currentTime-state.LastSaveTime > 2 // 2秒防抖
}

// UpdateLastSaveTime 更新最后保存时间
func (state *UIState) UpdateLastSaveTime() {
	state.LastSaveTime = time.Now().Unix()
}

// LayoutTitle 布局标题
func LayoutTitle(gtx layout.Context, th *material.Theme) layout.Dimensions {
	title := material.H3(th, "=== goxrdp Windows Remote Connection Tool ===")
	title.Alignment = text.Middle
	return title.Layout(gtx)
}

// LayoutIPInfo 布局IP信息
func LayoutIPInfo(gtx layout.Context, th *material.Theme, currentIP string) layout.Dimensions {
	ipLabel := material.Body1(th, "当前IP地址: "+currentIP)
	ipLabel.Alignment = text.Middle
	return ipLabel.Layout(gtx)
}

// LayoutFormFields 布局表单字段
func LayoutFormFields(gtx layout.Context, th *material.Theme, state *UIState) layout.Dimensions {
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(th, "远程服务器:").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Editor(th, &state.RemoteServerEntry, "远程服务器地址 (例如: https://piko-upstream.friddle.me:8082)").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(th, "连接选项:").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(th, &state.LocalControlCheck, "控制本地机器").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(th, "主机地址:").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Editor(th, &state.HostEntry, "输入要连接的远程主机IP地址").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(th, "用户名:").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Editor(th, &state.UsernameEntry, "输入用户名").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(th, "密码:").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Editor(th, &state.PasswordEntry, "输入密码").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(th, "连接名称:").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Editor(th, &state.ConnectionNameEntry, "连接名称 (默认为用户名)").Layout(gtx)
		}),
	)
}

// LayoutButtons 布局按钮
func LayoutButtons(gtx layout.Context, th *material.Theme, state *UIState) layout.Dimensions {
	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(th, &state.SaveConfigBtn, "保存配置").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(th, &state.ConnectBtn, "连接").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(th, &state.QuitBtn, "退出").Layout(gtx)
		}),
	)
}

// LayoutStatus 布局状态信息
func LayoutStatus(gtx layout.Context, th *material.Theme, state *UIState) layout.Dimensions {
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(th, "连接状态:").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if state.StatusText != "" {
				statusLabel := material.Body1(th, state.StatusText)
				statusLabel.Color = state.GetStatusColor()
				return statusLabel.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if state.IsConnected && state.AccessURL != "" {
				return material.Body1(th, "访问URL: "+state.AccessURL).Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if state.LastError != "" {
				errorLabel := material.Body1(th, "错误信息: "+state.LastError)
				errorLabel.Color = color.NRGBA{R: 255, G: 0, B: 0, A: 255}
				return errorLabel.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
	)
}
