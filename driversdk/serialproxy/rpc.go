// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package serialproxy

// 串口代理的线协议载荷：**插件 → 网关**的反向请求（pluginwire.MethodSerial*）。
//
// 信封与方向规则见 pluginwire.Frame 的注释。载荷方向与 HTTP 相反，容易看错：
// 请求由插件发、应答由网关回，两者共用同一个信封。
//
// 字节流用 []byte，JSON 编码下自动 base64；串口帧本身很小（Modbus 一帧最多
// 256 字节），为它引入一套二进制编码不值得 —— 代理路径的瓶颈在物理链路。

// SerialOpenParams serial.open 请求参数。
//
// 网关按「归一端口 + 串口参数」共享同一个物理句柄：两个插件进程请求同一端口
// 且参数一致时拿到各自的句柄 id，但底下是同一个 fd 与同一把跨进程总线锁。
// 参数不一致是配置错误，网关直接拒绝（同一条总线上的设备必须参数一致）。
type SerialOpenParams struct {
	Port     string `json:"port"`
	BaudRate int    `json:"baudRate"`
	DataBits int    `json:"dataBits"`
	StopBits int    `json:"stopBits"`
	Parity   string `json:"parity"`
}

// SerialOpenResult serial.open 应答。
type SerialOpenResult struct {
	// ID 网关侧的句柄 id（0 保留为"未分配"）。该句柄归属于调用它的插件进程，
	// 进程退出时网关自动关闭。
	ID uint64 `json:"id"`
}

// SerialHandleParams serial.close / serial.unlock 的参数。
type SerialHandleParams struct {
	ID uint64 `json:"id"`
}

// SerialLockParams serial.lock 的参数。
//
// TimeoutMs 是等待其它会话释放总线锁的上限。**提交请求的插件负责给出上限**：
// 一次长读取可能占着总线数十秒，无限等待会挂死插件的采集轮。
type SerialLockParams struct {
	ID        uint64 `json:"id"`
	TimeoutMs int    `json:"timeoutMs"`
}

// SerialWriteParams serial.write 的参数。
type SerialWriteParams struct {
	ID   uint64 `json:"id"`
	Data []byte `json:"data"`
}

// SerialWriteReply serial.write 应答。
type SerialWriteReply struct {
	N int `json:"n"`
}

// SerialReadParams serial.read 的参数。
//
// 网关在宿主侧最多等 TimeoutMs，一有字节就返回；超时且无字节时
// Reply.TimedOut=true —— 与本地串口 Read 的语义一致（部分字节 + nil error，
// 或无字节 + ErrTimeout）。
type SerialReadParams struct {
	ID        uint64 `json:"id"`
	Max       int    `json:"max"`
	TimeoutMs int    `json:"timeoutMs"`
}

// SerialReadReply serial.read 应答。
type SerialReadReply struct {
	Data     []byte `json:"data,omitempty"`
	TimedOut bool   `json:"timedOut,omitempty"`
}
