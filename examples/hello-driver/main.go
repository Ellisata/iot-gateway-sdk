// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

// Command iot-driver-hello is a minimal driver used as the SDK quick-start
// example. It does not talk to real hardware: Connect always succeeds and Read
// returns 42 for every point. Replace the body of Read with your protocol.
package main

import (
	"github.com/Ellisata/iot-gateway-sdk/driver"
	"github.com/Ellisata/iot-gateway-sdk/driversdk"
)

type helloDriver struct {
	connected bool
}

func (d *helloDriver) Connect(string) error { d.connected = true; return nil }
func (d *helloDriver) Ping(string) error    { return nil }
func (d *helloDriver) IsConnected() bool    { return d.connected }
func (d *helloDriver) Close() error         { d.connected = false; return nil }

func (d *helloDriver) Read(addrs []driver.DeviceAddress) ([]driver.ReadResult, error) {
	out := make([]driver.ReadResult, len(addrs))
	for i, a := range addrs {
		out[i] = driver.ReadResult{
			DeviceAddressID: a.ID,
			Value:           "42",
			DataType:        a.DataType,
			Kind:            driver.KindInt,
			Quality:         driver.QualityGood,
		}
	}
	return out, nil
}

func main() {
	driversdk.Main(
		driversdk.Factories{
			"Hello.TCP": func() driver.Driver { return &helloDriver{} },
		},
		driversdk.Meta{
			APIVersion:  driversdk.APIVersion,
			Scope:       "hello",
			Description: "Hello driver (SDK example)",
			Version:     "1.0.0",
			Protocols: []driversdk.Protocol{
				{Name: "Hello.TCP", Description: "Hello TCP"},
			},
			Types: []*driver.DataType{
				{Name: "int16", Kind: driver.KindInt, Size: 2},
			},
			TypeMap: map[string]string{
				driver.TypeShort: "int16",
			},
		},
		driversdk.Capabilities{},
	)
}
