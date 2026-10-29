# iot-gateway SDK

[简体中文](README.md) | English

**Write a protocol driver for iot-gateway — including a proprietary one — without
touching the AGPL-licensed gateway core.**

This repository is the driver SDK for
[iot-gateway](https://github.com/Ellisata/iot-gateway). It contains the driver
contract, the wire protocol, and the plugin serve loop. It is licensed under
**Apache-2.0**, does **not** import the AGPL gateway core, and its root module
has **zero third-party dependencies** — only the optional `testkit` module pulls
any.

## License boundary

| Component | License | What it means for you |
|---|---|---|
| `iot-gateway` gateway core | AGPL-3.0-only | Running it internally is fine; distributing a modified version triggers AGPL obligations. Commercial/OEM licensing is available. |
| **this SDK** | **Apache-2.0** | You may use it in proprietary drivers, redistribute it, and keep your driver source closed. |
| Your driver | your choice | It runs as a separate process and talks to the gateway over a documented wire protocol. It is an independent program. |

The SDK being Apache-2.0 is what makes the last row unambiguous: your driver
does not link or import AGPL code.

## Packages

| Package | Purpose |
|---|---|
| `driver` | Driver interface, data types, quality codes, type registry, discovery/point-scan contracts |
| `pluginwire` | Length-prefixed frame codec used between the gateway and a plugin process |
| `driversdk` | `Main`/`Serve`, plugin metadata, RPC parameter types, logging |
| `testkit` | Fake PLC servers for driver tests. **Nested module** in this repository (`testkit/go.mod`) — the only part that pulls third-party dependencies. |

## Test kit

`testkit` is a **nested module in this same repository**
(`github.com/Ellisata/iot-gateway-sdk/testkit`, with its own `go.mod`) — not a
separate repository and not a git submodule. Keeping it out of the root module
is what keeps the packages above dependency-free; keeping it in the same
repository lets fake servers and driver tests change together in one commit. It
provides in-process fake PLC servers that speak real TCP on a loopback port —
Modbus, S7, FINS, MC3E, DLT645, CIP and OPC UA — so a driver can be tested
end-to-end without hardware:

```bash
go get github.com/Ellisata/iot-gateway-sdk/testkit@v0.1.0
```

```go
import "github.com/Ellisata/iot-gateway-sdk/testkit/fake"

srv, err := fake.NewModbusTCP()
if err != nil { /* ... */ }
defer srv.Close()
// point your driver at srv.Addr()
```

The fakes are Apache-2.0 like the rest of the SDK. `testkit` is versioned
independently through prefixed tags (`testkit/vX.Y.Z`); see
[testkit/README.en.md](testkit/README.en.md#releasing) for the release rules.

## Quick start

```go
package main

import (
	"github.com/Ellisata/iot-gateway-sdk/driver"
	"github.com/Ellisata/iot-gateway-sdk/driversdk"
)

type helloDriver struct{ connected bool }

func (d *helloDriver) Connect(string) error           { d.connected = true; return nil }
func (d *helloDriver) Ping(string) error              { return nil }
func (d *helloDriver) IsConnected() bool              { return d.connected }
func (d *helloDriver) Close() error                   { d.connected = false; return nil }
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
		driversdk.Factories{"Hello.TCP": func() driver.Driver { return &helloDriver{} }},
		driversdk.Meta{
			APIVersion:  driversdk.APIVersion,
			Scope:       "hello",
			Description: "Hello driver",
			Protocols:   []driversdk.Protocol{{Name: "Hello.TCP"}},
			Types:       []*driver.DataType{{Name: "int16", Kind: driver.KindInt, Size: 2}},
			TypeMap:     map[string]string{driver.TypeShort: "int16"},
		},
		driversdk.Capabilities{},
	)
}
```

Build it as `iot-driver-hello` (`.exe` on Windows) and put it in the gateway's
`plugins/` directory:

```bash
go build -o plugins/iot-driver-hello .
```

The gateway loads it on start and hot-loads new plugin binaries at runtime.

## Compatibility

| SDK | Wire `APIVersion` | Gateways that accept it |
|---|---|---|
| v0.1.x | 4 | Any gateway that accepts `APIVersion == 4` (currently v0.2.0 and later) |

The wire protocol is versioned independently from the SDK's semver. A driver
built against SDK v0.1.x is compatible with any gateway that accepts
`APIVersion == 4`. A mismatch is rejected at handshake time, never silently
misinterpreted.

**The SDK version does not participate in the compatibility decision** — only
`APIVersion` does. Official releases pair a gateway version, a
`drivers.version` and an SDK version; the full pairing matrix, plugin
compatibility range and `APIVersion` bump rules are in
[Gateway version compatibility](https://github.com/Ellisata/iot-gateway/blob/main/docs/version-compatibility.md).

## Documentation

- [Driver development guide](docs/driver-development.md) (coming with v0.1.0)
- [Wire protocol](docs/wire-protocol.md) (coming with v0.1.0)
- Main project: [README](https://github.com/Ellisata/iot-gateway)

## Contributing

Contributions to this SDK are accepted under **Apache-2.0** with a
**Developer Certificate of Origin (DCO)** sign-off. No CLA is required.
See [CONTRIBUTING.en.md](CONTRIBUTING.en.md) and [DCO](DCO).

The SDK must never import the AGPL gateway core. CI enforces this with a
dependency-direction check.

## Security

A plugin is a native executable that runs with the gateway's OS privileges.
The SDK makes writing a driver easy; it does not sandbox it. Only install
plugins you trust, keep `plugins/` writable by administrators only, and
prefer signed/checksummed distributions in production.
