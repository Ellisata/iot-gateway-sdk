// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package driver

// DeviceAddress 是驱动 Read 的输入。
//
// 只有 ID / Name / DataType 会跨进程边界（见 driversdk.WireAddr）：插件进程
// 收到的是这三项，其余字段一律为零值。CommonDataType 只供**宿主侧**的
// NormalizeAddressType 使用，插件不应依赖它在 Read 里有值。
//
// 这个类型刻意不引用 iot-gateway 核心的 model/po.DeviceAddress：SDK 以
// Apache-2.0 独立发布，不能反向依赖 AGPL 的网关核心。宿主在调用驱动前把
// 数据库模型转换成这个类型。
type DeviceAddress struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DataType string `json:"dataType"`

	// CommonDataType 通用类型名（配置界面展示用），仅宿主侧使用。
	// 不参与线协议；插件收到的该字段恒为空。
	CommonDataType string `json:"commonDataType,omitempty"`
}
