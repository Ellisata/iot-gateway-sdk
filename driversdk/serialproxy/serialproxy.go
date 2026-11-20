// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

// Package serialproxy 是「网关代理串口」的插件侧实现。
//
// 两个**不同插件进程**配到同一个物理串口时（例如 DLT645.Serial 与 Modbus.RTU），
// 各自打开一个 fd 会同时踩三个坑：内核把每个字节只交给其中一个读端（应答被另一个
// 插件读走）、select 竞争失败方拿到 EAGAIN、任一方 Close 恢复 termios 打断另一方。
// 进程内的 sharing 修不了这一类 —— 正确的做法是网关主进程持有**唯一**句柄，插件
// 经 serial.* RPC 收发字节。
//
// 本包把这条 RPC 字节流包装成 io.ReadWriteCloser，并额外提供 Lock/Unlock：
// 它们转发到网关侧的跨进程总线锁。serialbus 会把资源上的 Lock/Unlock 串进自己的
// 总线锁，因此驱动代码完全不需要区分"本地串口"与"宿主代理"。
//
// 读语义与本地串口一致：一有字节就返回；超时且无字节时返回 ErrTimeout。
//
// Package serialproxy implements the plugin side of gateway-proxied serial I/O.
// It exposes the RPC byte stream as an io.ReadWriteCloser whose Lock/Unlock
// forward to the gateway's cross-process bus lock. Read semantics match a local
// serial port: return as soon as bytes arrive, ErrTimeout when nothing arrived
// within the driver's read timeout.
package serialproxy

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Ellisata/iot-gateway-sdk/driver"
	"github.com/Ellisata/iot-gateway-sdk/pluginwire"
)

// ErrTimeout 读超时且一个字节都没收到。驱动用它区分"设备没应答"与"链路坏了"。
//
// 刻意与 goburrow/serial 的 ErrTimeout 分开：SDK 根模块零第三方依赖。
// 驱动的超时判定必须同时认这两个（见各驱动的 isTimeoutErr / markBroken）。
var ErrTimeout = errors.New("serialproxy: read timeout")

// lockTimeout 等待网关总线锁的上限。
//
// 采集引擎已按物理端口把整轮串行化，正常路径几乎不会真的等锁；这个上限防的是
// 测试连接/点位扫描与一次长达数十秒的批量读取撞车时无限期挂死。
const lockTimeout = 10 * time.Second

// Caller 发起一次插件 → 网关的反向 RPC。由 driversdk.Serve 实现。
//
// 没有 ctx：调用方只在宿主进程存活期间调用，宿主下线时底层管道关闭，
// Call 会随之返回（见 driversdk 的 server.Call）。
type Caller interface {
	Call(method string, params, result any) error
}

// Open 请求网关打开（或共享）一个物理串口。
func Open(c Caller, opts driver.SerialOpenOptions) (io.ReadWriteCloser, error) {
	var res SerialOpenResult
	err := c.Call(pluginwire.MethodSerialOpen, SerialOpenParams{
		Port:     opts.Port,
		BaudRate: opts.BaudRate,
		DataBits: opts.DataBits,
		StopBits: opts.StopBits,
		Parity:   opts.Parity,
	}, &res)
	if err != nil {
		return nil, fmt.Errorf("serialproxy: open %s: %w", opts.Port, err)
	}
	if res.ID == 0 {
		return nil, fmt.Errorf("serialproxy: open %s: 网关返回了无效的句柄 id", opts.Port)
	}
	return &Conn{caller: c, id: res.ID, readTimeout: opts.ReadTimeout}, nil
}

// Conn 一条经网关代理的串口字节流。
type Conn struct {
	caller      Caller
	id          uint64
	readTimeout time.Duration

	mu     sync.Mutex
	err    error // 粘性错误：RPC 层失败或加锁失败后不再继续收发
	closed bool
	locked bool
}

// Read 读取已到达的字节；超时且无字节时返回 ErrTimeout。
func (c *Conn) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if err := c.stickyErr(); err != nil {
		return 0, err
	}

	timeout := c.readTimeout
	if timeout <= 0 {
		timeout = time.Second
	}
	var rep SerialReadReply
	if err := c.caller.Call(pluginwire.MethodSerialRead, SerialReadParams{
		ID:        c.id,
		Max:       len(p),
		TimeoutMs: durationMs(timeout),
	}, &rep); err != nil {
		c.fail(err)
		return 0, err
	}
	if len(rep.Data) == 0 {
		// 宿主回了空数据但没标超时，等价于"这段时间没有字节"。
		return 0, ErrTimeout
	}
	return copy(p, rep.Data), nil
}

// Write 发送字节。
func (c *Conn) Write(p []byte) (int, error) {
	if err := c.stickyErr(); err != nil {
		return 0, err
	}
	var rep SerialWriteReply
	if err := c.caller.Call(pluginwire.MethodSerialWrite, SerialWriteParams{
		ID:   c.id,
		Data: p,
	}, &rep); err != nil {
		c.fail(err)
		return 0, err
	}
	return rep.N, nil
}

// Close 释放网关侧的句柄。幂等。
func (c *Conn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	locked := c.locked
	c.locked = false
	c.mu.Unlock()

	if locked {
		_ = c.caller.Call(pluginwire.MethodSerialUnlock, SerialHandleParams{ID: c.id}, nil)
	}
	err := c.caller.Call(pluginwire.MethodSerialClose, SerialHandleParams{ID: c.id}, nil)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// Lock 获取网关侧的跨进程总线锁。
//
// 签名刻意不带 error：驱动的 Lock()/Unlock() 契约是 void（见 serialbus）。
// 加锁失败时把错误粘在 Conn 上，本次事务随后的 Read/Write 会直接返回它，
// 驱动据此把链路标为不可用并重连。
func (c *Conn) Lock() {
	if err := c.stickyErr(); err != nil {
		return
	}
	if err := c.caller.Call(pluginwire.MethodSerialLock, SerialLockParams{
		ID:        c.id,
		TimeoutMs: durationMs(lockTimeout),
	}, nil); err != nil {
		c.fail(err)
		return
	}
	c.mu.Lock()
	c.locked = true
	c.mu.Unlock()
}

// Unlock 释放网关侧的跨进程总线锁。
func (c *Conn) Unlock() {
	c.mu.Lock()
	locked := c.locked
	c.locked = false
	c.mu.Unlock()
	if !locked {
		return
	}
	_ = c.caller.Call(pluginwire.MethodSerialUnlock, SerialHandleParams{ID: c.id}, nil)
}

func (c *Conn) stickyErr() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("serialproxy: connection is closed")
	}
	return c.err
}

func (c *Conn) fail(err error) {
	c.mu.Lock()
	if c.err == nil {
		c.err = err
	}
	c.mu.Unlock()
}

func durationMs(d time.Duration) int {
	ms := d.Milliseconds()
	if ms <= 0 {
		return 1
	}
	return int(ms)
}
