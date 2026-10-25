// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package main

import (
	"testing"

	"github.com/Ellisata/iot-gateway-sdk/driver"
)

func TestHelloDriverRead(t *testing.T) {
	d := &helloDriver{}
	if err := d.Connect(`{"host":"127.0.0.1"}`); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if !d.IsConnected() {
		t.Fatal("IsConnected = false after Connect")
	}
	got, err := d.Read([]driver.DeviceAddress{
		{ID: "p1", Name: "point-1", DataType: "int16"},
		{ID: "p2", Name: "point-2", DataType: "int16"},
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	for i, r := range got {
		if r.DeviceAddressID == "" || r.Value != "42" || r.Quality != driver.QualityGood {
			t.Fatalf("result[%d] = %+v", i, r)
		}
	}
}
