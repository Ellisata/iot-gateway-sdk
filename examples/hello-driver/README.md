# Hello 驱动

[English](README.en.md) | 简体中文

最小的 iot-gateway 驱动。它声明一种协议（`Hello.TCP`），对每个点位都返回 `42`。

```bash
go build -o plugins/iot-driver-hello .
```

把二进制复制到网关的 `plugins/` 目录，然后创建协议为 `Hello.TCP` 的设备。
本示例只用于跑通构建；把 `Read` 替换成你真正的协议实现。

真实驱动保持相同结构，并补充：

- 在 `Connect` 中解析协议相关的配置；
- 重连与超时处理；
- 每个点位的质量码（`driver.Quality*`），而不是一律 `QualityGood`；
- 用假设备编写测试（CI 无需 PLC）。
