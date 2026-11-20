// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package serialproxy

import (
	"errors"
	"testing"
	"time"

	"github.com/Ellisata/iot-gateway-sdk/driver"
	"github.com/Ellisata/iot-gateway-sdk/pluginwire"
)

// fakeCaller 模拟网关侧的 serial.* 处理。
type fakeCaller struct {
	openID   uint64
	readData []byte
	timedOut bool
	readErr  error

	calls    []string
	writes   [][]byte
	closed   bool
	locked   bool
	unlocked bool
	lockErr  error
}

func (f *fakeCaller) Call(method string, params, result any) error {
	f.calls = append(f.calls, method)
	switch method {
	case pluginwire.MethodSerialOpen:
		*result.(*SerialOpenResult) = SerialOpenResult{ID: f.openID}
	case pluginwire.MethodSerialLock:
		if f.lockErr != nil {
			return f.lockErr
		}
		f.locked = true
	case pluginwire.MethodSerialUnlock:
		f.unlocked = true
	case pluginwire.MethodSerialWrite:
		f.writes = append(f.writes, append([]byte(nil), params.(SerialWriteParams).Data...))
		*result.(*SerialWriteReply) = SerialWriteReply{N: len(params.(SerialWriteParams).Data)}
	case pluginwire.MethodSerialRead:
		if f.readErr != nil {
			return f.readErr
		}
		*result.(*SerialReadReply) = SerialReadReply{Data: f.readData, TimedOut: f.timedOut}
	case pluginwire.MethodSerialClose:
		f.closed = true
	}
	return nil
}

func openTestConn(t *testing.T, f *fakeCaller) *Conn {
	t.Helper()
	f.openID = 7
	conn, err := Open(f, driver.SerialOpenOptions{
		Port: "COM1", BaudRate: 2400, DataBits: 8, StopBits: 1, Parity: "E",
		ReadTimeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	return conn.(*Conn)
}

// 读：有数据原样返回；无数据且宿主标了超时，翻译成 ErrTimeout。
func TestReadSemantics(t *testing.T) {
	f := &fakeCaller{readData: []byte{0x11, 0x22}}
	conn := openTestConn(t, f)

	buf := make([]byte, 8)
	n, err := conn.Read(buf)
	if err != nil || n != 2 || buf[0] != 0x11 || buf[1] != 0x22 {
		t.Fatalf("Read = (%d, %v) buf=% X", n, err, buf[:n])
	}

	f.readData = nil
	f.timedOut = true
	if _, err := conn.Read(buf); !errors.Is(err, ErrTimeout) {
		t.Fatalf("超时读返回值 %v，want ErrTimeout", err)
	}
}

// Lock/Unlock 必须转到宿主；加锁失败时后续读写立刻返回粘性错误。
func TestLockFailureIsSticky(t *testing.T) {
	f := &fakeCaller{lockErr: errors.New("bus busy")}
	conn := openTestConn(t, f)

	conn.Lock() // 失败不返回错误（签名是 void），粘在 Conn 上
	conn.Unlock()

	if _, err := conn.Read(make([]byte, 4)); err == nil {
		t.Fatal("加锁失败后 Read 仍成功 —— 会在没有总线锁的情况下收发")
	}
	if _, err := conn.Write([]byte{1}); err == nil {
		t.Fatal("加锁失败后 Write 仍成功")
	}
}

func TestLockUnlockForwarded(t *testing.T) {
	f := &fakeCaller{}
	conn := openTestConn(t, f)

	conn.Lock()
	if !f.locked {
		t.Fatal("Lock 未转发到宿主")
	}
	conn.Unlock()
	if !f.unlocked {
		t.Fatal("Unlock 未转发到宿主")
	}
}

func TestWriteAndClose(t *testing.T) {
	f := &fakeCaller{}
	conn := openTestConn(t, f)

	n, err := conn.Write([]byte{1, 2, 3})
	if err != nil || n != 3 {
		t.Fatalf("Write = (%d, %v)", n, err)
	}
	if len(f.writes) != 1 || string(f.writes[0]) != string([]byte{1, 2, 3}) {
		t.Fatalf("宿主收到的数据 = % X", f.writes)
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("Close 失败: %v", err)
	}
	if !f.closed {
		t.Fatal("Close 未转发到宿主")
	}
	// 幂等
	if err := conn.Close(); err != nil {
		t.Fatalf("重复 Close 失败: %v", err)
	}
}

func TestOpenRejectsZeroID(t *testing.T) {
	f := &fakeCaller{openID: 0}
	if _, err := Open(f, driver.SerialOpenOptions{Port: "COM1"}); err == nil {
		t.Fatal("宿主返回 0 号句柄时应当报错")
	}
}
