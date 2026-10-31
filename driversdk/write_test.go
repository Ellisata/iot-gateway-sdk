// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package driversdk_test

import (
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Ellisata/iot-gateway-sdk/driver"
	"github.com/Ellisata/iot-gateway-sdk/driversdk"
	"github.com/Ellisata/iot-gateway-sdk/pluginwire"
)

// writableStubDriver 在 stubDriver 上补一个可编程的 Write。
type writableStubDriver struct {
	*stubDriver

	mu       sync.Mutex
	writes   []driver.WriteRequest
	writeErr error
}

func (d *writableStubDriver) Write(reqs []driver.WriteRequest) ([]driver.WriteResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.writes = append(d.writes, reqs...)
	if d.writeErr != nil {
		return nil, d.writeErr
	}
	out := make([]driver.WriteResult, len(reqs))
	for i, req := range reqs {
		out[i] = driver.WriteResult{
			DeviceAddressID: req.DeviceAddressID,
			Code:            driver.WriteCodeOK,
			Value:           req.Value,
		}
	}
	return out, nil
}

func writableTestMeta() driversdk.Meta {
	m := testMeta()
	m.Protocols[0].Writable = true
	return m
}

func TestServeWrite(t *testing.T) {
	drv := &writableStubDriver{stubDriver: &stubDriver{}}
	conn := startServer(t, func() driver.Driver { return drv }, writableTestMeta())
	c := newRPCClient(t, conn)

	var cr driversdk.ConnectResult
	c.callOK(pluginwire.MethodConnect, driversdk.ConnectParams{Protocol: testProtocol}, &cr)

	reqs := []driver.WriteRequest{
		{DeviceAddressID: "a1", Name: "4x0001", DataType: "int16", Value: "42"},
		{DeviceAddressID: "a2", Name: "0x0001", DataType: "bool", Value: "true"},
	}
	var reply driversdk.WriteReply
	c.callOK(pluginwire.MethodWrite, driversdk.WriteParams{Session: cr.Session, Writes: reqs}, &reply)

	if !reply.Connected {
		t.Errorf("WriteReply.Connected = false，期望 true")
	}
	if len(reply.Results) != len(reqs) {
		t.Fatalf("结果数 = %d，期望 %d", len(reply.Results), len(reqs))
	}
	for i, got := range reply.Results {
		if got.DeviceAddressID != reqs[i].DeviceAddressID || !got.OK() || got.Value != reqs[i].Value {
			t.Errorf("结果[%d] = %+v，期望 id=%s value=%s code=OK",
				i, got, reqs[i].DeviceAddressID, reqs[i].Value)
		}
	}

	drv.mu.Lock()
	gotWrites := len(drv.writes)
	drv.mu.Unlock()
	if gotWrites != len(reqs) {
		t.Errorf("驱动收到 %d 条写入，期望 %d", gotWrites, len(reqs))
	}
}

// 协议没有声明 Writable 时，即使驱动实现了 Writer 也必须拒绝。
// 这条防线的意义是把"网关能力表"作为唯一开关，而不是依赖驱动内部实现。
func TestServeWriteProtocolNotWritable(t *testing.T) {
	drv := &writableStubDriver{stubDriver: &stubDriver{}}
	conn := startServer(t, func() driver.Driver { return drv }, testMeta())
	c := newRPCClient(t, conn)

	var cr driversdk.ConnectResult
	c.callOK(pluginwire.MethodConnect, driversdk.ConnectParams{Protocol: testProtocol}, &cr)

	f := c.call(pluginwire.MethodWrite, driversdk.WriteParams{
		Session: cr.Session,
		Writes:  []driver.WriteRequest{{DeviceAddressID: "a1", Name: "4x0001", DataType: "int16", Value: "1"}},
	})
	if f.Error == nil || !strings.Contains(f.Error.Message, "Writable") {
		t.Fatalf("期望 Writable 拒绝，得到 %+v", f.Error)
	}

	drv.mu.Lock()
	defer drv.mu.Unlock()
	if len(drv.writes) != 0 {
		t.Errorf("驱动不应收到写入，实际 %d 条", len(drv.writes))
	}
}

// 声明 Writable 但工厂返回的实例没有实现 Writer，必须在启动期失败。
func TestServeWritableRequiresWriter(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		done <- driversdk.ServeConn(server, testFactories(func() driver.Driver {
			return &stubDriver{}
		}), writableTestMeta(), driversdk.Capabilities{})
	}()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "driver.Writer") {
			t.Fatalf("期望启动失败并提到 driver.Writer，得到 %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeConn 未在 2s 内返回")
	}
}

// 驱动返回链路错误时，错误必须过线，且连接状态一并回传。
func TestServeWriteTransportError(t *testing.T) {
	drv := &writableStubDriver{stubDriver: &stubDriver{connected: true}, writeErr: errors.New("boom")}
	conn := startServer(t, func() driver.Driver { return drv }, writableTestMeta())
	c := newRPCClient(t, conn)

	var cr driversdk.ConnectResult
	c.callOK(pluginwire.MethodConnect, driversdk.ConnectParams{Protocol: testProtocol}, &cr)

	f := c.call(pluginwire.MethodWrite, driversdk.WriteParams{
		Session: cr.Session,
		Writes:  []driver.WriteRequest{{DeviceAddressID: "a1", Name: "4x0001", DataType: "int16", Value: "1"}},
	})
	if f.Error == nil || !strings.Contains(f.Error.Message, "boom") {
		t.Fatalf("期望链路错误过线，得到 %+v", f.Error)
	}
	if len(f.Result) == 0 {
		t.Fatal("链路错误时也应回传 WriteReply（含 connected）")
	}
	var reply driversdk.WriteReply
	if err := pluginwire.DecodeBody(f.Codec, f.Result, &reply); err != nil {
		t.Fatalf("解码 WriteReply: %v", err)
	}
	if !reply.Connected {
		t.Error("WriteReply.Connected = false，期望 true")
	}
}
