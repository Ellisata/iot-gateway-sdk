// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package driversdk

import (
	"net"
	"testing"
	"time"

	"github.com/Ellisata/iot-gateway-sdk/driversdk/serialproxy"
	"github.com/Ellisata/iot-gateway-sdk/pluginwire"
)

// newHostCallServer 造一个最小 server：只有反向调用需要的字段，
// 并用一个读协程把网关应答交给 deliverHostReply（与 run 的路由一致）。
func newHostCallServer(t *testing.T) (*server, net.Conn) {
	t.Helper()
	srvConn, gwConn := net.Pipe()
	s := &server{
		r:           srvConn,
		w:           srvConn,
		hostPending: make(map[uint64]chan pluginwire.Frame),
		done:        make(chan struct{}),
	}
	go func() {
		for {
			f, err := pluginwire.ReadFrame(srvConn)
			if err != nil {
				return
			}
			s.deliverHostReply(f)
		}
	}()
	t.Cleanup(func() {
		s.stopOnce.Do(func() { close(s.done) })
		srvConn.Close()
		gwConn.Close()
	})
	return s, gwConn
}

// TestHostCallRoundTrip 插件 → 网关的反向调用能正确编码请求、解码应答。
func TestHostCallRoundTrip(t *testing.T) {
	s, gwConn := newHostCallServer(t)

	go func() {
		f, err := pluginwire.ReadFrame(gwConn)
		if err != nil {
			return
		}
		if f.Method != pluginwire.MethodSerialRead {
			t.Errorf("网关收到方法 %q, want %q", f.Method, pluginwire.MethodSerialRead)
			return
		}
		var p serialproxy.SerialReadParams
		if err := pluginwire.DecodeBody(f.Codec, f.Params, &p); err != nil {
			t.Errorf("解码参数失败: %v", err)
			return
		}
		if p.Max != 16 || p.TimeoutMs != 25 {
			t.Errorf("参数 = %+v, want Max=16 Timeout=25", p)
		}
		body, codec, _ := pluginwire.EncodeBody(serialproxy.SerialReadReply{Data: []byte{0xAA}})
		_ = pluginwire.WriteFrame(gwConn, pluginwire.Frame{ID: f.ID, Codec: codec, Result: body})
	}()

	var rep serialproxy.SerialReadReply
	if err := s.Call(pluginwire.MethodSerialRead, serialproxy.SerialReadParams{
		ID: 3, Max: 16, TimeoutMs: 25,
	}, &rep); err != nil {
		t.Fatalf("Call 失败: %v", err)
	}
	if len(rep.Data) != 1 || rep.Data[0] != 0xAA {
		t.Fatalf("应答 = % X", rep.Data)
	}
}

// TestHostCallRemoteError 网关返回的错误要原样传给调用方。
func TestHostCallRemoteError(t *testing.T) {
	s, gwConn := newHostCallServer(t)

	go func() {
		f, err := pluginwire.ReadFrame(gwConn)
		if err != nil {
			return
		}
		_ = pluginwire.WriteFrame(gwConn, pluginwire.Frame{
			ID:    f.ID,
			Error: &pluginwire.RemoteError{Message: "串口被占用"},
		})
	}()

	err := s.Call(pluginwire.MethodSerialLock, serialproxy.SerialLockParams{ID: 1}, nil)
	if err == nil || err.Error() != "串口被占用" {
		t.Fatalf("Call 返回 %v, want 网关的错误文本", err)
	}
}

// TestHostCallStopsWhenClosed 网关连接断开时，等待中的反向调用必须立即返回，
// 不能挂在一个永远不来的应答上。
func TestHostCallStopsWhenClosed(t *testing.T) {
	s, _ := newHostCallServer(t)
	s.stopOnce.Do(func() { close(s.done) })

	done := make(chan error, 1)
	go func() {
		done <- s.Call(pluginwire.MethodSerialRead, serialproxy.SerialReadParams{ID: 1}, nil)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("连接已关闭时 Call 仍成功")
		}
	case <-time.After(time.Second):
		t.Fatal("连接已关闭时 Call 未返回")
	}
}
