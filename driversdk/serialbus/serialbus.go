// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

// Package serialbus 在一个插件进程内按「物理端口」共享串口句柄与总线锁。
//
// 一个插件进程里可能同时有**多台设备**（多个会话）配在同一个物理串口上 ——
// RS-485 挂多台从站是常态（Modbus RTU 靠站号、DL/T 645 靠表号寻址）。如果每个
// 会话各开一个句柄，会同时踩三个坑，三个都会让整条总线上的设备一起瘫痪：
//
//  1. **同一个 tty 出现两个读端**，内核把每个字节只交付给其中一个 fd。A 的设备
//     应答被 B 读走，A 从此收不到 → 超时、残缺帧（日志里的"已收 7 字节"）。
//  2. 非阻塞 fd + select 让两个读端同时被唤醒、同时 read，竞争失败的一方拿到
//     EAGAIN（"resource temporarily unavailable"），并被判为链路断开。
//  3. Linux 的 termios 是 **tty 级**状态，goburrow/serial 在 Close 时恢复一份
//     打开前的备份 —— 一个会话重连关闭句柄，会把另一个仍开着的句柄所在端口改回
//     cooked/echo 模式，于是对方也判死、也重连，两边交替"连接已关闭/已连接"。
//
// 正确模型是**一个物理端口、一个句柄、一个总线锁**：所有会话 Acquire 到同一个
// Handle（它的 Lock 就是总线锁），最后一个会话 Release 时才真正关闭端口。
//
// 串口参数（波特率/数据位/停止位/校验位）是 tty 级配置，不同参数的设备**不能**
// 共享句柄；这种配置本身就是错误的（同一条线上的设备必须参数一致），必须明确
// 报错，而不是再开一个句柄把上面三个坑原样复现。
//
// 本包只依赖标准库与 driver 包，不认识任何具体协议：资源类型由调用方通过类型
// 参数给出 —— 直接读写的驱动用 io.ReadWriteCloser，封装了协议陪跑对象的驱动
// （例如 Modbus RTU 的 handler）用自己的类型，只要实现 io.Closer。
//
// 已知边界：注册表是**进程级**的。不同插件的进程（例如 DLT645.Serial 与
// Modbus.RTU 配到同一个端口）无法共享句柄 —— 那种配置仍会复现上述问题，网关侧
// 只能按总线把收发错开并给出告警（见网关 collector 的总线锁键），真正的解决需要
// 一个端口代理，不在本包范围内。
//
// 典型用法：
//
//	var ports = serialbus.New[io.ReadWriteCloser]()
//
//	h, err := ports.Acquire(serialbus.Key{Port: cfg.ComPort, Baud: 9600, DataBits: 8, StopBits: 1, Parity: "N"},
//		func() (io.ReadWriteCloser, error) { return openSerial(cfg) })
//	if err != nil { return err }
//	defer h.Release() // 最后一个引用才真正关端口
//
//	h.Lock()
//	defer h.Unlock()
//	_, _ = h.Resource().Write(frame)
//	...
package serialbus

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Ellisata/iot-gateway-sdk/driver"
)

// Key 一个串口句柄的身份：端口名 + 影响 tty 配置的参数。
//
// 传入的 Port 与 Parity 会在 Acquire 里归一（Port 走 driver.CanonicalSerialPort，
// Parity 统一大写）—— 调用方可以放心把配置里的原值直接填进来。
//
// 参数必须由调用方套好**默认值**再传入：不同的协议默认值不同（DL/T 645 是
// 2400/E，Modbus RTU 与 FINS 是 9600/N），本包不知道也不该猜。省略与显式写出
// 默认值必须得到同一个 Key，否则两台配置等价、写法不同的设备会被判成参数冲突
// —— 或者更糟，各开一个句柄。
//
// 超时不参与 Key：它是单次读的等待时间，不是总线的电气参数；共享句柄时以
// 最先打开它的那次配置为准。
type Key struct {
	Port     string
	Baud     int
	DataBits int
	StopBits int
	Parity   string
}

// String 返回参数部分的可读形式（不含端口名），供日志与冲突报错使用。
func (k Key) String() string {
	return fmt.Sprintf("baud=%d,data=%d,stop=%d,parity=%s",
		k.Baud, k.DataBits, k.StopBits, k.Parity)
}

// normalize 归一端口名与校验位大小写。参数的默认值由调用方负责（见 Key 的说明）。
func (k Key) normalize() Key {
	k.Port = driver.CanonicalSerialPort(k.Port)
	k.Parity = strings.ToUpper(strings.TrimSpace(k.Parity))
	return k
}

// Bus 一个进程内的端口注册表。每个驱动包持有自己的实例（不同协议的句柄类型
// 无法共用一张表），实例之间彼此独立。
//
// 必须用 New 创建；零值不可用。
type Bus[T io.Closer] struct {
	mu     sync.Mutex
	byKey  map[Key]*Handle[T]
	byPort map[string]map[Key]*Handle[T]
}

// New 创建一张空的端口注册表。
func New[T io.Closer]() *Bus[T] {
	return &Bus[T]{
		byKey:  make(map[Key]*Handle[T]),
		byPort: make(map[string]map[Key]*Handle[T]),
	}
}

func (b *Bus[T]) addLocked(h *Handle[T]) {
	b.byKey[h.key] = h
	set := b.byPort[h.key.Port]
	if set == nil {
		set = make(map[Key]*Handle[T], 1)
		b.byPort[h.key.Port] = set
	}
	set[h.key] = h
}

func (b *Bus[T]) removeLocked(h *Handle[T]) {
	if cur, ok := b.byKey[h.key]; ok && cur == h {
		delete(b.byKey, h.key)
	}
	set := b.byPort[h.key.Port]
	if set == nil {
		return
	}
	delete(set, h.key)
	if len(set) == 0 {
		delete(b.byPort, h.key.Port)
	}
}

// liveOnPortLocked 返回该端口上参数不同的、仍可用的句柄（没有则 nil）。
func (b *Bus[T]) liveOnPortLocked(port string) *Handle[T] {
	for _, h := range b.byPort[port] {
		if h.live.Load() {
			return h
		}
	}
	return nil
}

// Acquire 取得该配置对应的串口句柄：能复用就复用（引用计数 +1），否则调用
// open 打开并登记。
//
// open 在注册表锁内置调用，因此同一端口不会被并发打开两次；open 失败不会在
// 注册表里留下任何条目。open 不得返回 (nil, nil)：成功时必须交出可用的资源。
//
// 同一端口、参数不同时返回错误而不是打开第二个句柄：共享一套参数会让另一台
// 设备的每一帧都是错的，各开一个句柄则会复现包注释里的三个坑。这种配置本身
// 就是错的，必须让现场看见。
func (b *Bus[T]) Acquire(key Key, open func() (T, error)) (*Handle[T], error) {
	key = key.normalize()
	if key.Port == "" {
		return nil, fmt.Errorf("串口名未配置")
	}

	b.mu.Lock()

	if h := b.byKey[key]; h != nil {
		if h.live.Load() {
			h.refs++
			b.mu.Unlock()
			return h, nil
		}
		// 句柄已判死（链路上的 I/O 错误被驱动记下）：摘登记、关旧句柄，再走下面
		// 的新开路径。关闭放在注册表锁内做，保证下一次 open 一定发生在旧 fd
		// 关闭（termios 已恢复）之后 —— 否则新句柄备份到的 termios 会被旧的
		// Close 覆盖。等待一次在途交互的时间受协议单帧超时约束，重连本就不
		// 频繁，可以接受。
		b.removeLocked(h)
		h.released = true
		_ = h.closeLocked()
	}

	if other := b.liveOnPortLocked(key.Port); other != nil {
		otherParams := other.key.String()
		b.mu.Unlock()
		return nil, fmt.Errorf("串口 %s 已被另一台设备以不同参数打开（%s），"+
			"本设备为（%s）；同一条总线上的设备必须使用相同的串口参数",
			key.Port, otherParams, key.String())
	}

	res, err := open()
	if err != nil {
		b.mu.Unlock()
		return nil, err
	}

	h := &Handle[T]{bus: b, key: key, res: res, refs: 1}
	h.live.Store(true)
	b.addLocked(h)
	b.mu.Unlock()
	return h, nil
}

// Handle 一个共享的串口句柄。多个会话持有的 Handle 可能是同一个对象 ——
// 它们共享同一条总线、同一把锁、同一个连接状态。
type Handle[T io.Closer] struct {
	mu sync.Mutex // 总线锁：一次「发送 → 读应答」全程持有

	bus  *Bus[T]
	key  Key
	res  T
	live atomic.Bool // 句柄是否可用；置 false 后 Acquire 会换新句柄

	// 以下字段只由 Bus.mu 保护。
	refs     int
	released bool
	closed   bool
}

// Lock 取得总线锁。一次完整的请求-应答必须全程持有它，否则同一条总线上的
// 两个会话会在物理线路上交错，破坏帧结构。
//
// 资源如果还实现 Lock()/Unlock()（代理串口的 Conn 就是如此，它转发到网关的
// **跨进程**总线锁），这里会一并取得。顺序固定为进程内锁 → 跨进程锁，所有
// 会话方向一致，不会死锁。
func (h *Handle[T]) Lock() {
	h.mu.Lock()
	if l, ok := any(h.res).(interface{ Lock() }); ok {
		l.Lock()
	}
}

// Unlock 释放总线锁（与 Lock 对称：先跨进程锁，再进程内锁）。
func (h *Handle[T]) Unlock() {
	if l, ok := any(h.res).(interface{ Unlock() }); ok {
		l.Unlock()
	}
	h.mu.Unlock()
}

// Resource 返回底层资源（串口或协议 handler）。
//
// 调用方必须持有 Lock，并且先确认 Alive —— 判死后底层资源可能已被回收。
// 这里刻意不加锁：加锁会与调用方已持有的总线锁自锁。
func (h *Handle[T]) Resource() T { return h.res }

// Key 返回归一后的身份（端口名与参数），供日志与诊断使用。
func (h *Handle[T]) Key() Key { return h.key }

// Alive 报告这个共享句柄是否仍可用。链路上出现非超时的 I/O 错误后，驱动应调用
// MarkBroken；下一次 Acquire 会换一个新句柄。
func (h *Handle[T]) Alive() bool {
	return h != nil && h.live.Load()
}

// MarkBroken 标记共享句柄已失效。任何持有它的会话都可以调；标记之后本句柄上
// 的收发都应视为失败，直到有会话重新 Acquire 到新句柄。
func (h *Handle[T]) MarkBroken() {
	if h == nil {
		return
	}
	h.live.Store(false)
}

// Refs 返回当前的引用数（打开的会话数）。只用于日志与诊断，数值随时可能变化。
func (h *Handle[T]) Refs() int {
	h.bus.mu.Lock()
	defer h.bus.mu.Unlock()
	return h.refs
}

// Release 释放本会话对句柄的引用；最后一个引用才真正关闭底层资源。
// 重复调用是安全的（幂等）。
func (h *Handle[T]) Release() error {
	if h == nil || h.bus == nil {
		return nil
	}
	b := h.bus

	b.mu.Lock()
	if h.released {
		b.mu.Unlock()
		return nil
	}
	if h.refs > 1 {
		h.refs--
		b.mu.Unlock()
		return nil
	}
	b.removeLocked(h)
	h.released = true
	err := h.closeLocked()
	b.mu.Unlock()
	return err
}

// closeLocked 真正关闭底层资源。调用方保证这是最后一个引用（或该句柄已被判死
// 并被摘除）。持总线锁关闭，等一次在途交互结束，避免在别人收发到一半时把
// fd 抽走。
func (h *Handle[T]) closeLocked() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.live.Store(false)
	if h.closed {
		return nil
	}
	h.closed = true
	return h.res.Close()
}
