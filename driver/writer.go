// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package driver

// 写（控制下发）相关的契约。
//
// 写能力是**可选**的：driver.Driver 仍然只有 Connect/Ping/Read/IsConnected/Close。
// 需要控制下发的协议额外实现 Writer，并在 driversdk.Meta.Protocols[].Writable
// 里声明。这样新增写能力不会破坏任何既有插件，也不需要递增 APIVersion
// （与 SerialExclusive / ConnectionMatcher 的取舍一致）。
//
// 与 Read 的对称性：单点失败进 WriteResult.Code，只有**链路级**失败才让
// Write 返回 error。返回 error 时调用方必须把整批写入视为"状态未知"，并
// 通过 Read 回读确认实际结果 —— PLC 的批量写不是事务。

// 写结果码。
const (
	// WriteCodeOK 写入成功（设备已接受该写请求）。
	WriteCodeOK = 0
	// WriteCodeBadAddress 地址无法解析，或该地址在设备上不可写。
	WriteCodeBadAddress = 1
	// WriteCodeNotWritable 该点位/区域本身是只读的（如 Modbus 输入寄存器、
	// S7 输入区），与地址是否合法无关。
	WriteCodeNotWritable = 2
	// WriteCodeBadValue 值无法按声明的数据类型编码（格式错误、越界、类型不支持）。
	WriteCodeBadValue = 3
	// WriteCodeUnsupported 当前协议/驱动不支持写入。
	WriteCodeUnsupported = 4
	// WriteCodeRejected 设备明确拒绝了这次写入（协议层否定应答）。
	WriteCodeRejected = 5
	// WriteCodeTransport 链路级失败：这次写入是否到达设备、是否生效都未知。
	// 与 Write 返回 error 同时出现，调用方应整批按"未知"处理并回读确认。
	WriteCodeTransport = 6
)

// WriteRequest 一次单点写入请求。
//
// DataType 是**协议内部类型名**（与 Read 的 DeviceAddress.DataType 同源），
// 驱动据此在插件内完成编码 —— DataType.Encode 不出插件进程，网关侧拿不到
// 编解码函数，因此值必须以字符串形式过线，由驱动解析后编码。
type WriteRequest struct {
	// DeviceAddressID 点位 id，用于把结果与请求一一对应。
	DeviceAddressID string `json:"deviceAddressId"`
	// Name 协议地址（如 DB1.DBW0、4x0001）。
	Name string `json:"name"`
	// DataType 协议内部类型名（如 int16、float32）。
	DataType string `json:"dataType"`
	// Value 待写入值的字符串形式。按 DataType 的 Kind 解析（bool/int/uint/float/string/time）。
	Value string `json:"value"`
}

// WriteResult 单点写入结果。
type WriteResult struct {
	DeviceAddressID string `json:"deviceAddressId"`
	// Code 写结果码，WriteCodeOK 表示成功。
	Code int `json:"code"`
	// Message 失败原因（成功时应为空）。
	Message string `json:"message,omitempty"`
	// Value 驱动实际写入的值（按 DataType.Format 格式化）。
	// 可为空：部分协议/类型没有回读能力，调用方不应依赖它判断成功。
	Value string `json:"value,omitempty"`
}

// OK 报告该点是否写入成功。
func (r WriteResult) OK() bool { return r.Code == WriteCodeOK }

// Writer 由支持控制下发的驱动实现。
//
// 实现约束：
//   - reqs 可能包含多个点位；**批量写不保证原子性**，逐点结果放 WriteResult。
//   - 单点失败（地址非法、只读、值非法、设备拒绝）返回 WriteResult（Code != 0），
//     不要返回 error；只有链路/会话级失败才返回 error。
//   - 返回值必须与 reqs 按 DeviceAddressID 一一对应，且每个 id 恰好出现一次。
//   - 驱动必须自行串行化对底层连接的访问（插件会话锁已提供一层，但驱动内部
//     的位读改写等操作仍需保证完整）。
type Writer interface {
	Write(reqs []WriteRequest) ([]WriteResult, error)
}
