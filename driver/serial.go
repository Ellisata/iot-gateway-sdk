// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package driver

import (
	"encoding/json"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SerialPortField 串口名在 protocol_json 中的字段名。**跨协议约定**：
// 所有串口类协议（SerialExclusive 为 true 的那些）都必须使用这个键。
//
// 为什么它是网关侧的契约而不是各插件内部的事：串口是独占资源，"测试连接"
// 失败时网关要能回答"这个口当前被哪台设备的采集任务占着"，而网关口拿到的
// 只有 protocol_json —— 它不认识任何具体协议，只能靠这个约定好的键取值。
//
// 违反约定的症状很隐蔽：占用提示**永远不出现**，且没有任何日志或报错，
// 因为 ExtractSerialPort 取不到值只会返回空串。故该约定由
// docs/driver-plugins.md §3.6 与各插件的表单契约测试共同钉住。
const SerialPortField = "comPort"

// SerialOpenFailureHint 串口打开失败时的统一提示，由各串口驱动在**打开失败的分支里**
// 拼到错误后面。
//
// 为什么这句话必须由驱动来说：网关口分不清"打不开端口"和"打开了但没应答" ——
// 两者在它眼里都是一条 RPC 错误。而这句话只对前者成立，挂到后者上会把现场引向
// "去关别的程序"，实际问题却在设备或链路参数上。驱动正站在打开失败的分支里，
// 它知道得最准。
//
// 另一个原因是可读性：底层抛上来的是 `Access is denied.`（在中文 Windows 上
// 还是"拒绝访问。"），既不说明是什么操作失败，也不给任何可照做的动作。
//
// 网关侧还会在此基础上补出**具体是哪个设备的采集任务占着**（见 service 的占用
// 提示）；两句都是真的，叠在一起也读得通，只是那种情况本就少见 ——
// 网关自己的设备占着端口时，"测试连接"正常应当走复用成功。
const SerialOpenFailureHint = "（串口打开失败：请核对端口名是否正确，或该端口已被其它程序占用）"

// CanonicalSerialPort 把指向同一物理端口的不同写法归一。
//
// 为什么必须是**同一个**函数：网关按归一后的端口名把设备归并到同一把总线锁，
// 插件进程按归一后的端口名共享同一个句柄。两边各写一份规则，迟早会出现
// "网关认为两台设备在同一条总线上、插件却为它们开了两个句柄"——症状正是本仓库
// 要修的那个（应答字节被瓜分、EAGAIN、交替重连），而且从日志看不出来。
//
// 规则：
//   - 两端空白与空串：裁掉空白，空串原样返回空串；
//   - Windows：`\\.\COM3` 去掉前缀，名字统一大写（串口名大小写不敏感）；
//   - Linux/macOS：绝对路径解析软链（`/dev/serial/by-id/...` 与 `/dev/ttyUSB0`
//     是同一个口）。相对路径不解析：同一个名字会在网关与插件各自的 cwd 下
//     展开成不同路径，解析后反而会把同一个口归一成两个名字。
//
// 归一失败（路径不存在、无权限）时退回原串：不能因为规范化失败把设备当成
// 没有端口，真正的打开错误由驱动在打开分支里报。
func CanonicalSerialPort(port string) string {
	p := strings.TrimSpace(port)
	if p == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		p = strings.TrimPrefix(p, `\\.\`)
		return strings.ToUpper(p)
	}
	if filepath.IsAbs(p) {
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			return resolved
		}
	}
	return p
}

// SerialOpenOptions 打开一个串口所需的全部参数。
//
// 它同时服务两条路径：驱动自己打开本机串口，以及"网关代理串口"——
// 两个插件进程配到同一个物理口时，由网关主进程持有唯一句柄，插件经 RPC
// 收发字节（见 SerialTransportProvider）。
type SerialOpenOptions struct {
	Port     string
	BaudRate int
	DataBits int
	StopBits int
	Parity   string

	// ReadTimeout 单次 Read 的阻塞上限（驱动自己的读超时）。
	//
	// 代理路径上它决定一次 serial.read RPC 最多等多久；直连路径上它被写进
	// 串口的打开参数。**必须由驱动给出**：DL/T 645 用 20ms 做轮询友好的短读，
	// Modbus/FINS/MC 用配置的帧超时，两者不可互换。
	ReadTimeout time.Duration
}

// SerialTransportOpener 打开一个可读写的串口字节流。
//
// 返回的实现如果还提供 `Lock() / Unlock()`，serialbus 会把它串进总线锁：
// 代理路径上这两个方法转发到网关的跨进程总线锁，直连路径上不存在。
// 这样驱动代码无需区分自己拿到的是本地串口还是远端代理。
type SerialTransportOpener func(SerialOpenOptions) (io.ReadWriteCloser, error)

// SerialTransportProvider 由需要访问串口的驱动实现（可选接口）。
//
// SDK 在串口代理模式下通过它注入了 opener；**opener 为 nil 表示直接打开本机
// 串口**。驱动应把 opener 存在实例上，并在建立串口传输时优先使用它 ——
// 一个驱动实例（= 一台设备）在代理模式下必须全程走宿主，不能自己再开一个 fd。
//
// 为什么是可选接口而不是 driver.Driver 的新方法：加了新方法会让全部旧插件
// 在编译期失效，而这里只需要少数串口驱动参与。协议在 Meta 里声明
// `SerialProxy: true` 且驱动未实现本接口时，Serve 启动即失败。
type SerialTransportProvider interface {
	SetSerialTransport(opener SerialTransportOpener)
}

// SerialProxyCapable 可选接口：宿主侧驱动用它报告"本协议支持网关代理串口"。
//
// 网关只有在同一条物理端口上的**全部**协议都返回 true 时才会启用代理；
// 混入一个不支持代理的插件时保持各自的直连并告警（那一档仍会互相干扰）。
type SerialProxyCapable interface {
	SupportsSerialProxy() bool
}

// SerialProxySetter 可选接口：网关在建立任务/会话前通知驱动启用代理。
//
// 必须在 Connect 之前调用；驱动的下一次 Connect 会经由注入的
// SerialTransportOpener 拿字节流，而不是自己打开端口。
type SerialProxySetter interface {
	SetSerialProxy(enabled bool)
}

// ExtractSerialPort 从 protocol_json 中取出串口名。
//
// 不存在、不是字符串、或去掉空白后为空 → 返回 ""。
//
// 键名比较用 EqualFold，与 encoding/json 把 JSON 键匹配到 `json:"comPort"`
// 结构体字段时的大小写不敏感行为对齐 —— 否则会出现「驱动收得下 ComPort、
// 占用提示却看不见它」这种自相矛盾的状态。
//
// "我在测哪个口"与"谁占着这个口"两条路径必须调用同一个函数，提示才自洽。
func ExtractSerialPort(protocolJSON string) string {
	if protocolJSON == "" {
		return ""
	}
	// 用 RawMessage 而不是 map[string]any：后者会把数字、布尔也解出来，
	// 而这里只认字符串，多一步类型判断没有意义。
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(protocolJSON), &fields); err != nil {
		return ""
	}
	for k, raw := range fields {
		if !strings.EqualFold(k, SerialPortField) {
			continue
		}
		var port string
		if err := json.Unmarshal(raw, &port); err != nil {
			return ""
		}
		return strings.TrimSpace(port)
	}
	return ""
}
