# 更新日志

本文件记录 iot-gateway SDK 的所有重要变更，遵循[语义化版本](https://semver.org/)。每个版本先列中文条目，再列英文条目。线协议有自己的 `APIVersion`；兼容性对照表见 README.md。

All notable changes to the iot-gateway SDK are documented here, following
[Semantic Versioning](https://semver.org/). Chinese entries come first under
each release, followed by English entries. The wire protocol has its own
`APIVersion`; see the compatibility table in README.md.

## [0.6.0] - 2026-11-24

### 新增

- `driver.DeviceAddress.ByteOrderLayout` 与 `driver.WriteRequest.ByteOrderLayout`：
  点位级字节/字排列布局（如 Modbus 的 `ABCD` / `CDAB` / `BADC` / `DCBA`，64 位为
  `ABCDEFGH` / `CDABGHEF` / …）。空串 = 跟随设备配置（`protocol_json`），这是旧
  点位的唯一行为。布局按**值宽度**解释，同一布局名在 16/32/64 位下展开成不同的
  字节排列；只有「同一设备上不同点位可能采用不同厂商约定」的协议（如 Modbus）
  用得到，其余协议忽略。读写必须用同一份值，否则会出现「读回来是对的、写下去
  是错的」。
- `driversdk.WireAddr.ByteOrderLayout`，以及 read 入参的位置式二进制载荷。
  解码发生在插件进程里，网关侧没有解码函数，因此该字段必须过线。
- `driver.RangeSig` 现在把 `ByteOrderLayout` 纳入指纹。缓存的规划里保留着构建
  时那份 addrs，解析直接读它 —— 指纹若不覆盖布局，单改点位布局会命中旧规划、
  按旧布局解码。新增 `TestRangeSigIncludesByteOrder` 守住这一点。

### 兼容性

- 线协议 `APIVersion` **4 → 5**。`ReadParams` 是位置式二进制载荷（不是 JSON），
  旧插件会把新增的字符串当成额外尾随字节并在 `Done()` 处报错，因此**必须整体
  重编**；新插件在旧网关上握手阶段即被拒绝。没有采用「旧插件忽略未知字段」的
  兼容路线：静默按设备级配置解码会采出看似合理、实为错位排列的值，比直接拒绝
  更糟。帧魔数依旧保持 `IGP2`。
- 不使用点位级布局的插件行为不变：`ByteOrderLayout` 为空串时完全等同旧逻辑，
  改动的只是协议版本号。

### Added

- `driver.DeviceAddress.ByteOrderLayout` and `driver.WriteRequest.ByteOrderLayout`:
  a point-level byte/word layout (e.g. Modbus `ABCD` / `CDAB` / `BADC` / `DCBA`;
  64-bit `ABCDEFGH` / `CDABGHEF` / …). An empty string means "follow the device
  configuration" (`protocol_json`), which is the only behavior old points had. The
  layout is interpreted per value width — the same name expands to different byte
  arrangements at 16/32/64 bits — and is only meaningful for protocols where points
  on one device may follow different vendor conventions (Modbus); other protocols
  ignore it. Read and write must use the same value, or a point reads back right
  and writes wrong.
- `driversdk.WireAddr.ByteOrderLayout`, carried in the positional binary payload
  of read requests. Decoding happens in the plugin process and the gateway has no
  decode function, so the field must cross the wire.
- `driver.RangeSig` now folds `ByteOrderLayout` into the fingerprint. A cached
  plan keeps the `addrs` it was built from and parsing reads them directly, so a
  fingerprint that ignored the layout would hit the old plan and decode with the
  old layout. `TestRangeSigIncludesByteOrder` guards this.

### Compatibility

- Wire `APIVersion` goes **4 → 5**. `ReadParams` is a positional binary payload
  (not JSON): an old plugin treats the new string as extra trailing bytes and
  fails at `Done()`, so both sides **must be rebuilt**; a new plugin is rejected at
  handshake by an old gateway. The "old plugin ignores unknown fields" route was
  deliberately not taken — silently decoding with the device-level configuration
  yields plausible-looking but misordered values, which is worse than a hard
  refusal. Frame magic stays `IGP2`.
- Plugins that never use point-level layouts are behaviorally unchanged: an empty
  `ByteOrderLayout` is exactly the old logic; only the protocol version moves.

## [0.5.0] - 2026-11-20

### 新增

- `driver.CanonicalSerialPort`：串口名归一的唯一实现 —— 两端空白、Windows 的
  大小写与 `\\.\` 前缀、Linux `/dev/serial/by-id/...` 软链。网关按归一后的端口名
  归并总线锁，插件按归一后的端口名共享句柄；两边必须用同一个函数，否则会出现
  "网关认为同总线、插件却开了两个句柄"的静默故障。
- `driversdk/serialbus`：进程级串口句柄共享注册表。同一物理端口、相同串口参数的
  多个会话 `Acquire` 到同一个 `Handle`，它的 `Lock` 就是总线锁，最后一个
  `Release` 才真正关闭端口；参数不同明确报配置错误；句柄判死后下一次 `Acquire`
  自动换新。资源类型由调用方用类型参数给出（裸串口用 `io.ReadWriteCloser`，
  协议陪跑对象用自己的类型），包本身只依赖标准库与 `driver`。

- `driversdk/serialproxy` 与反向 RPC：两个**不同插件进程**配到同一个物理口时，
  由网关主进程持有唯一句柄，插件经 `pluginwire.MethodSerial*`（serial.open /
  lock / read / write / unlock / close）收发字节。`serialproxy.Conn` 把它们包装成
  `io.ReadWriteCloser`，并转发 Lock/Unlock 到网关的跨进程总线锁。
- `driver.SerialOpenOptions` / `driver.SerialTransportProvider`：串口驱动通过这个
  **可选接口**接收 SDK 注入的字节流 opener（nil = 直接打开本机串口）。
- `driversdk.Protocol.SerialProxy` 能力声明；`ConnectParams` / `PingParams` /
  `DiscoverPointsParams` 新增 `serialProxy` 标志。`serialbus.Handle` 现在会把
  资源上的 Lock/Unlock 串进总线锁（代理流借此获得跨进程互斥）。

### 兼容性

- 线协议 `APIVersion` **保持 4**。代理能力通过 `serialProxy` 标志协商：旧网关
  从不置位，旧插件不声明 `SerialProxy`，双方都走原来的直连路径；`serial.*`
  反向方法只在网关主动启用代理时才会出现。需要多个会话共用同一串口的插件应当
  迁移到 `serialbus`，否则仍会复现"应答字节被瓜分 / EAGAIN / 交替重连"。

### Added

- `driver.CanonicalSerialPort`: the single implementation of serial port name
  canonicalization — surrounding whitespace, Windows case and the `\\.\` prefix,
  Linux `/dev/serial/by-id/...` symlinks. The gateway groups devices into one bus
  lock by the canonical name and plugins share handles by it; both sides must use
  the same function, or you get a silent "gateway thinks one bus, plugin opens two
  handles" split.
- `driversdk/serialbus`: a process-wide registry that shares one serial handle
  across all sessions on the same physical port with the same line parameters.
  `Handle.Lock` is the bus lock, the port is closed only on the last `Release`, a
  parameter mismatch is an explicit configuration error, and a dead handle is
  replaced by the next `Acquire`. The resource type is a type parameter (raw ports
  use `io.ReadWriteCloser`, protocol handlers use their own type); the package
  depends only on the standard library and `driver`.

- `driversdk/serialproxy` and reverse RPC: when two **different plugin processes**
  share one physical port, the gateway process owns the only handle and plugins
  exchange bytes over `pluginwire.MethodSerial*` (`serial.open` / `lock` / `read`
  / `write` / `unlock` / `close`). `serialproxy.Conn` wraps them as an
  `io.ReadWriteCloser` and forwards Lock/Unlock to the gateway's cross-process bus
  lock.
- `driver.SerialOpenOptions` / `driver.SerialTransportProvider`: serial drivers
  receive an injected byte-stream opener through this **optional interface**
  (nil = open the local port directly).
- `driversdk.Protocol.SerialProxy` capability declaration; `ConnectParams`,
  `PingParams` and `DiscoverPointsParams` gain a `serialProxy` flag.
  `serialbus.Handle` now composes a resource's Lock/Unlock into its bus lock,
  which is how proxied streams share cross-process mutual exclusion.

### Compatibility

- Wire `APIVersion` **stays 4**. Proxying is negotiated by the `serialProxy`
  flag: old gateways never set it and old plugins never declare `SerialProxy`,
  so both keep the direct path; the `serial.*` reverse methods only appear when
  the gateway deliberately enables proxying. Plugins that put several sessions on
  one serial port should migrate to `serialbus`, otherwise they still hit split
  answer bytes, `EAGAIN`, and alternating reconnects.

## [0.4.0] - 2026-11-05

### 新增

- `driver.ProbeTransport` 与 `driver.DiscoverySpec.Transport`：插件在 describe
  时声明单播 probe 使用 `tcp` / `udp` / 未声明。网关据此决定能否在 probe 之前
  做一次短超时的 TCP 可达性预筛（大网段扫描的主要加速手段）；广播目标永远绕过。
- `driversdk.Meta.Validate` 拒绝 `Transport` 的未知取值（只接受空串 / `tcp` / `udp`），
  把拼错的声明挡在插件启动期。

### 兼容性

- 线协议 `APIVersion` **保持 4**。新字段是可选且 `omitempty`：旧网关忽略未知字段，
  旧插件不带该字段时为零值（不做预筛）。旧插件二进制无需重编。

### Added

- `driver.ProbeTransport` and `driver.DiscoverySpec.Transport`: plugins declare
  whether their unicast probe uses `tcp`, `udp`, or leaves it unspecified. The
  gateway uses it to decide whether a short-timeout TCP reachability prefilter
  can run before the probe (the main speed lever for large-subnet scans);
  broadcast targets always bypass it.
- `driversdk.Meta.Validate` rejects unknown `Transport` values (only empty /
  `tcp` / `udp`), catching typos at plugin startup.

### Compatibility

- Wire `APIVersion` **stays 4**. The field is optional and `omitempty`: an old
  gateway ignores it, and an old plugin's zero value disables the prefilter.
  Old plugin binaries need no rebuild.

## [0.3.0] - 2026-10-31

### 新增

- `driver.Writer`：可选的控制下发接口，与 `SerialExclusive` / `ConnectionMatcher`
  同为**可选接口**，`driver.Driver` 本身不变。`WriteRequest` / `WriteResult` /
  `WriteCodeXxx` 定义在 `driver/writer.go`。
- `driversdk.Protocol.Writable`：协议级写能力声明。声明为 true 但工厂未返回
  `driver.Writer` 时 `Serve` 启动即失败。
- `pluginwire.MethodWrite` + `driversdk.WriteParams/WriteReply`：写走现有会话，
  载荷保持 JSON（冷路径）；插件与 Read 共用会话锁。
- testkit 的假 Modbus 从站现在会更新寄存器/线圈状态，供写后读回用例使用。

### 兼容性

- 线协议 `APIVersion` **保持 4**。新增方法/字段都是增量的：新网关 + 旧插件走
  `Writable=false` 分支（不开放写），旧网关 + 新插件忽略未知字段且从不调用
  `MethodWrite`。因此无需重编旧插件。

### Added

- `driver.Writer`: optional control-write interface, in the same style as
  `SerialExclusive` / `ConnectionMatcher`; `driver.Driver` is unchanged.
  `WriteRequest` / `WriteResult` / `WriteCodeXxx` live in `driver/writer.go`.
- `driversdk.Protocol.Writable`: protocol-level write capability. If declared
  true but the factory does not return a `driver.Writer`, `Serve` fails at startup.
- `pluginwire.MethodWrite` + `driversdk.WriteParams/WriteReply`: writes reuse the
  existing session, keep JSON payloads (cold path), and share the plugin session
  lock with `Read`.
- The testkit fake Modbus slave now updates register/coil state, enabling
  write-then-read tests.

### Compatibility

- Wire `APIVersion` **stays 4**. The new method/fields are additive: a new gateway
  with an old plugin sees `Writable=false` and does not expose writes; an old
  gateway with a new plugin ignores the unknown field and never calls
  `MethodWrite`. Old plugin binaries need no rebuild.

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
