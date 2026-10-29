# 更新日志

本文件记录 iot-gateway SDK 的所有重要变更，遵循[语义化版本](https://semver.org/)。每个版本先列中文条目，再列英文条目。线协议有自己的 `APIVersion`；兼容性对照表见 README.md。

All notable changes to the iot-gateway SDK are documented here, following
[Semantic Versioning](https://semver.org/). Chinese entries come first under
each release, followed by English entries. The wire protocol has its own
`APIVersion`; see the compatibility table in README.md.

## [0.2.0] - 2026-10-29

### 新增

- `driver.DiscoverySpec` / `driver.PointDiscoverySpec`：可选的 `TitleI18n` 和
  `DescriptionI18n` 映射。键名遵循 form-create 的 `options.language` 约定
  （`zh-cn` / `en`）；缺少某个语言时回退到 `Title` / `Description`。

### 兼容性

- 线协议 `APIVersion` 保持 4。新字段是可选且 omitempty 的：旧网关会忽略它们，
  旧插件不填（nil），网关回退到原先的中文字符串。

### Added

- `driver.DiscoverySpec` / `driver.PointDiscoverySpec`: optional `TitleI18n` and
  `DescriptionI18n` maps. Keys follow form-create's `options.language` convention
  (`zh-cn` / `en`); a missing locale falls back to `Title` / `Description`.

### Compatibility

- Wire `APIVersion` stays 4. The new fields are optional and omitempty: old
  gateways ignore them, old plugins leave them nil, and the gateway falls back to
  the original Chinese strings.

## [0.1.0] - 2026-10-25

### 新增

- 首个独立 SDK，从 iot-gateway 核心中抽取。
- `driver`：驱动契约、数据类型、质量码、类型注册表、设备/点位发现契约。
- `pluginwire`：带长度前缀的帧编解码（`FrameMagic` = `IGP2`）。
- `driversdk`：`Main`/`Serve`、插件元数据、RPC 参数类型、stderr 日志。
- Apache-2.0 授权，DCO 贡献政策。
- 零第三方依赖；不导入 AGPL 网关核心。

### 兼容性

- 线协议 `APIVersion`：4。
- 网关：v0.2.x 及以后，直到下一次协议进位。

### Added

- Initial standalone SDK, extracted from the iot-gateway core.
- `driver`: driver contract, data types, quality codes, type registry,
  device/point discovery contracts.
- `pluginwire`: length-prefixed frame codec (`FrameMagic` = `IGP2`).
- `driversdk`: `Main`/`Serve`, plugin metadata, RPC parameter types, stderr
  logging.
- Apache-2.0 licensing, DCO contribution policy.
- Zero third-party dependencies; no import of the AGPL gateway core.

### Compatibility

- Wire `APIVersion`: 4.
- Gateway: v0.2.x and later, until the next protocol bump.
