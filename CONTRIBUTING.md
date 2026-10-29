# 为 iot-gateway SDK 贡献代码

[English](CONTRIBUTING.en.md) | 简体中文

## 贡献的许可

本 SDK 以 **Apache-2.0** 授权。提交贡献即表示你同意你的贡献以 Apache-2.0
授权（入向 = 出向）。**无需 CLA。**

每个提交都必须带有 **DCO（开发者原创声明）** 签署：

```bash
git commit -s -m "driver: add ..."
```

该签署确认 [DCO](DCO) 中的声明。缺少匹配 `Signed-off-by:` 尾注的提交，其
PR 会被 CI 拒绝。

## 硬性规则

1. **绝不导入 AGPL 网关核心。** SDK 必须始终可供私有驱动使用。CI 会执行
   依赖方向检查；引入 `iot-gateway` 导入的 PR 会失败。
2. **未经讨论不引入第三方运行时依赖。** SDK 刻意只依赖标准库，使驱动除 Go
   工具链之外没有供应链面。
3. **线协议兼容性是契约。** 不要修改 JSON 字段名、帧布局或二进制载荷顺序，
   除非同时提升 `driversdk.APIVersion` 并记录迁移。Golden/parity 测试必须
   保持通过。
4. **每个新文件都必须带 Apache-2.0 SPDX 头：**

   ```go
   // SPDX-License-Identifier: Apache-2.0
   // Copyright (c) 2026 Edwin and iot-gateway contributors
   ```

## 开发

```bash
go test ./...
gofmt -l .
go vet ./...
```

测试套件只依赖标准库，无需网关或 PLC 即可运行。

## 报告安全问题

请勿为漏洞开公开 issue。遵循主仓库的安全策略（SECURITY.md），或私下联系维护者。
