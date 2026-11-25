// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package driver

// DeviceAddress 是驱动 Read 的输入。
//
// ID / Name / DataType / ByteOrderLayout 会跨进程边界（见
// driversdk.WireAddr）：插件进程收到的是这四项，其余字段一律为零值。
// CommonDataType 只供**宿主侧**的 NormalizeAddressType 使用，插件不应依赖它
// 在 Read 里有值。
//
// 这个类型刻意不引用 iot-gateway 核心的 model/po.DeviceAddress：SDK 以
// Apache-2.0 独立发布，不能反向依赖 AGPL 的网关核心。宿主在调用驱动前把
// 数据库模型转换成这个类型。
type DeviceAddress struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DataType string `json:"dataType"`

	// ByteOrderLayout 点位级字节/字排列布局（如 ABCD / CDAB / BADC / DCBA，
	// 64 位为 ABCDEFGH / CDABGHEF / …）。空串 = **跟随设备配置**
	// （protocol_json），这是绝大多数点位的取值，也是旧插件的唯一行为。
	//
	// 布局按**值宽度**解释：同一个布局名在 16/32/64 位下展开成不同的字节排列
	// （见各驱动的布局表）。目前 Modbus 与欧姆龙 FINS 实现了该字段（FINS 的
	// 字符串、bool、8 位类型与位地址不适用）；其余协议忽略它，宿主也会在保存时
	// 拒绝给它们配置非空布局。驱动应把它当作 DECODE 约定的一部分：读与写
	// （driver.WriteRequest 的同名字段）必须使用同一份值，否则会出现
	// 「读对一个、写错一个」。
	ByteOrderLayout string `json:"byteOrderLayout,omitempty"`

	// CommonDataType 通用类型名（配置界面展示用），仅宿主侧使用。
	// 不参与线协议；插件收到的该字段恒为空。
	CommonDataType string `json:"commonDataType,omitempty"`
}
