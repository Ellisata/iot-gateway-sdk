# testkit —— 驱动测试用的假 PLC 服务器

[English](README.en.md) | 简体中文

iot-gateway SDK 的可选配套模块。它提供进程内的假 PLC 服务器（Modbus、S7、FINS、MC3E、DLT645、CIP、OPC UA），在回环端口上说真实 TCP，因此驱动无需硬件即可端到端测试。

它有意做成**独立的 Go 模块**：SDK 主模块（`github.com/Ellisata/iot-gateway-sdk`）保持零依赖，而 `testkit` 为 OPC UA 假服务器引入 `gopcua`。只在需要这些假服务器时引入：

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

与 SDK 一样采用 Apache-2.0 授权。

## 所在位置

`testkit` **不是**独立仓库，也**不是** git submodule。它是 [iot-gateway-sdk 仓库](https://github.com/Ellisata/iot-gateway-sdk)内的嵌套 Go 模块，与 SDK 共享同一份 git 历史——没有 `.gitmodules` 条目，也没有第二个 `.git` 目录。

这个安排的两半都是有意为之：

- **是嵌套模块，但不并进根模块。** 根模块必须保持零依赖（见 [CONTRIBUTING](../CONTRIBUTING.md) 中的硬性规则 2）；只有 `testkit` 引入 `gopcua`。导入 `driver` 或 `driversdk` 永远不会把它拖进来。
- **在同一仓库，但不拆出去。** 假服务器编码的协议行为会与驱动及其测试一起演进，因此假服务器的修复和驱动的修复在同一次提交中落地，并由同一次 CI 验证。拆出去会强制跨仓按固定顺序升版本，中间态主干是坏的；还会改变模块路径，破坏所有既有的 `.../testkit/fake` 导入。

请不要把 `testkit` vendor 进根模块，也不要为它创建独立仓库。

## 发布

`testkit` 通过**带前缀的 git tag** 独立于 SDK 发版：

| 改动的范围 | 需要打的 tag | 使用方式 |
|---|---|---|
| 仅 `testkit/` | `testkit/v0.2.0` | `go get github.com/Ellisata/iot-gateway-sdk/testkit@v0.2.0` |
| `testkit/` 以外的任何内容 | `v0.2.0` | `go get github.com/Ellisata/iot-gateway-sdk@v0.2.0` |
| 两者都有 | 两个 tag | 上述两种 |

`testkit/` 前缀只存在于 git tag；你 require 的模块版本仍是普通的 `v0.2.0`。CI 通过后从 `main` 手工打 tag；`testkit` 没有独立的发布流水线。

## 开发

在本目录下运行工具——根模块的 `go test ./...` 不覆盖 `testkit`，CI 会用独立的 job 测试它：

```bash
cd testkit
gofmt -l .
go vet ./...
go test -count=1 ./...
```

在 SDK 与驱动仓库之间联调时，把两个模块都纳入 workspace，而不是手工改 `go.mod`：

```bash
go work use ./iot-gateway-sdk ./iot-gateway-sdk/testkit ./iot-gateway-drivers
```
