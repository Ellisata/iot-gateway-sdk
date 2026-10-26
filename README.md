# iot-gateway SDK

[English](README.en.md) | 简体中文

**无需接触 AGPL 授权的网关核心，就能为 iot-gateway 编写协议驱动——包括私有协议驱动。**

本仓库是 [iot-gateway](https://github.com/Ellisata/iot-gateway) 的驱动 SDK，包含驱动契约、线协议和插件的服务循环。它采用 **Apache-2.0** 授权，**不**导入 AGPL 网关核心，且根模块**零第三方依赖**——只有可选的 `testkit` 模块会引入依赖。

## 许可证边界

| 组件 | 许可证 | 对你的含义 |
|---|---|---|
| `iot-gateway` 网关核心 | AGPL-3.0-only | 内部使用没问题；分发修改版会触发 AGPL 义务。可提供商业/OEM 授权。 |
| **本 SDK** | **Apache-2.0** | 可用于私有驱动、可再分发，驱动源码可以闭源。 |
| 你的驱动 | 由你决定 | 它作为独立进程运行，通过有文档的线协议与网关通信，是独立程序。 |

SDK 采用 Apache-2.0，最后一行的结论才没有歧义：你的驱动不链接、也不导入 AGPL 代码。

## 包

| 包 | 用途 |
|---|---|
| `driver` | 驱动接口、数据类型、质量码、类型注册表、发现/点位扫描契约 |
| `pluginwire` | 网关与插件进程之间的长度前缀帧编解码 |
| `driversdk` | `Main`/`Serve`、插件元数据、RPC 参数类型、日志 |
| `testkit` | 驱动测试用的假 PLC 服务器。本仓库内的**嵌套模块**（`testkit/go.mod`）——唯一会引入第三方依赖的部分。 |

## 测试工具

`testkit` 是**同一仓库内的嵌套模块**（`github.com/Ellisata/iot-gateway-sdk/testkit`，自带 `go.mod`）——不是独立仓库，也不是 git submodule。把它排除在根模块之外，正是上面这些包保持零依赖的原因；把它留在同一仓库，则让假服务器和驱动测试能在一次提交中同步修改。它提供在回环端口上说真实 TCP 的进程内假 PLC 服务器——Modbus、S7、FINS、MC3E、DLT645、CIP 和 OPC UA——因此驱动无需硬件即可端到端测试：

```bash
go get github.com/Ellisata/iot-gateway-sdk/testkit@v0.1.0
```

```go
import "github.com/Ellisata/iot-gateway-sdk/testkit/fake"

srv, err := fake.NewModbusTCP()
if err != nil { /* ... */ }
defer srv.Close()
// point your driver at srv.Addr()
```

这些假服务器与 SDK 其余部分一样采用 Apache-2.0。`testkit` 通过前缀 tag（`testkit/vX.Y.Z`）独立发版；发布规则见 [testkit/README.md](testkit/README.md#发布)。

## 快速开始

```go
package main

import (
	"github.com/Ellisata/iot-gateway-sdk/driver"
	"github.com/Ellisata/iot-gateway-sdk/driversdk"
)

type helloDriver struct{ connected bool }

func (d *helloDriver) Connect(string) error           { d.connected = true; return nil }
func (d *helloDriver) Ping(string) error              { return nil }
func (d *helloDriver) IsConnected() bool              { return d.connected }
func (d *helloDriver) Close() error                   { d.connected = false; return nil }
func (d *helloDriver) Read(addrs []driver.DeviceAddress) ([]driver.ReadResult, error) {
	out := make([]driver.ReadResult, len(addrs))
	for i, a := range addrs {
		out[i] = driver.ReadResult{
			DeviceAddressID: a.ID,
			Value:           "42",
			DataType:        a.DataType,
			Kind:            driver.KindInt,
			Quality:         driver.QualityGood,
		}
	}
	return out, nil
}

func main() {
	driversdk.Main(
		driversdk.Factories{"Hello.TCP": func() driver.Driver { return &helloDriver{} }},
		driversdk.Meta{
			APIVersion:  driversdk.APIVersion,
			Scope:       "hello",
			Description: "Hello driver",
			Protocols:   []driversdk.Protocol{{Name: "Hello.TCP"}},
			Types:       []*driver.DataType{{Name: "int16", Kind: driver.KindInt, Size: 2}},
			TypeMap:     map[string]string{driver.TypeShort: "int16"},
		},
		driversdk.Capabilities{},
	)
}
```

构建为 `iot-driver-hello`（Windows 上是 `.exe`），放进网关的 `plugins/` 目录：

```bash
go build -o plugins/iot-driver-hello .
```

网关启动时会加载它，并支持运行时热加载新的插件二进制。

## 兼容性

| SDK | 线协议 `APIVersion` | 可加载的网关 |
|---|---|---|
| v0.1.x | 4 | 任何接受 `APIVersion == 4` 的网关（当前为 v0.2.0 及以后） |

线协议独立于 SDK 的 semver 进行版本管理。基于 SDK v0.1.x 构建的驱动，兼容任何接受 `APIVersion == 4` 的网关。不匹配会在握手阶段被拒绝，绝不会被静默误解析。

**SDK 版本号不参与兼容判定**：决定插件能否加载的只有 `APIVersion`。官方发行包按「网关版本 + `drivers.version` + SDK 版本」成对发布；完整配对矩阵、插件兼容区间与 `APIVersion` 进位规则见[网关版本兼容性](https://github.com/Ellisata/iot-gateway/blob/main/docs/version-compatibility.md)。

## 文档

- [驱动开发指南](docs/driver-development.md)（随 v0.1.0 提供）
- [线协议](docs/wire-protocol.md)（随 v0.1.0 提供）
- 主项目：[README](https://github.com/Ellisata/iot-gateway)

## 贡献

本 SDK 的贡献在 **Apache-2.0** 下接受，需 **DCO（开发者原创声明）** 签署。无需 CLA。参见 [CONTRIBUTING.md](CONTRIBUTING.md) 和 [DCO](DCO)。

SDK 绝不能导入 AGPL 网关核心。CI 通过依赖方向检查强制执行这一点。

## 安全

插件是以网关 OS 权限运行的原生可执行文件。SDK 让编写驱动变得容易，但它不提供沙箱。只安装你信任的插件，`plugins/` 仅允许管理员写入，生产环境优先使用已签名/带校验和的发行包。
