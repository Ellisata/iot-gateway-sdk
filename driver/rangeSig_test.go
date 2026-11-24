// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package driver_test

import (
	"testing"

	"github.com/Ellisata/iot-gateway-sdk/driver"
)

// 缓存的规划里保留着构建时那份 addrs，解析直接读它 —— 因此指纹必须覆盖
// 所有影响解码的字段。只改 ByteOrderLayout 却命中旧缓存，症状是
// 「改了点位字节序没生效，仍按旧布局（设备级）解码」。
func TestRangeSigIncludesByteOrder(t *testing.T) {
	base := []driver.DeviceAddress{
		{ID: "a", Name: "%MW0", DataType: "float32"},
		{ID: "b", Name: "%MW2", DataType: "float32"},
	}
	sameContent := []driver.DeviceAddress{
		{ID: "a", Name: "%MW0", DataType: "float32"},
		{ID: "b", Name: "%MW2", DataType: "float32"},
	}
	if driver.RangeSig(base) != driver.RangeSig(sameContent) {
		t.Fatal("相同批次内容应得到相同指纹")
	}

	orderChanged := append([]driver.DeviceAddress(nil), base...)
	orderChanged[1].ByteOrderLayout = "CDAB"
	if driver.RangeSig(base) == driver.RangeSig(orderChanged) {
		t.Fatal("仅改变点位布局必须换指纹，否则会命中旧规划按旧布局解码")
	}
}
