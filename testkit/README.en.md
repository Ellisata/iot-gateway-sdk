# testkit — fake PLC servers for driver tests

[简体中文](README.md) | English

Optional companion module to the iot-gateway SDK. It provides in-process fake PLC
servers (Modbus, S7, FINS, MC3E, DLT645, CIP, OPC UA) that speak real TCP on a
loopback port, so a driver can be tested end-to-end without hardware.

This is a **separate Go module** on purpose: the main SDK module
(`github.com/Ellisata/iot-gateway-sdk`) stays dependency-free, while `testkit`
pulls in `gopcua` for the OPC UA fake. Import it only if you need the fakes:

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

Licensed under Apache-2.0, same as the SDK.

## Where it lives

`testkit` is **not** a separate repository and **not** a git submodule. It is a
nested Go module inside the
[iot-gateway-sdk repository](https://github.com/Ellisata/iot-gateway-sdk),
tracked in the same git history as the SDK — there is no `.gitmodules` entry and
no second `.git` directory.

Both halves of that arrangement are deliberate:

- **Nested module, not merged into the root module.** The root module must stay
  dependency-free (see hard rule 2 in [CONTRIBUTING](../CONTRIBUTING.md)); only
  `testkit` pulls `gopcua`. Importing `driver` or `driversdk` never drags it in.
- **Same repository, not split out.** The fakes encode protocol behavior that
  evolves together with the drivers and their tests, so a fake fix and the
  driver fix land in one commit and are verified by one CI run. Splitting it out
  would force cross-repo version bumps in a fixed order and leave `main` broken
  in between. It would also change the module path and break every existing
  `.../testkit/fake` import.

Please do not vendor `testkit` into the root module and do not create a separate
repository for it.

## Releasing

`testkit` versions independently from the SDK through a **prefixed git tag**:

| Change touches | Tag to cut | Consumed as |
|---|---|---|
| `testkit/` only | `testkit/v0.2.0` | `go get github.com/Ellisata/iot-gateway-sdk/testkit@v0.2.0` |
| Anything outside `testkit/` | `v0.2.0` | `go get github.com/Ellisata/iot-gateway-sdk@v0.2.0` |
| Both | both tags | both of the above |

The `testkit/` prefix is part of the git tag only — the module version you
require is plain `v0.2.0`. Tags are cut by hand from `main` after CI is green;
there is no separate release pipeline for `testkit`.

## Development

Run tools from this directory — the root module's `go test ./...` does not cover
`testkit`, and CI tests it in its own job:

```bash
cd testkit
gofmt -l .
go vet ./...
go test -count=1 ./...
```

When working across the SDK and a driver repository, put both modules in the
workspace rather than hand-editing `go.mod`:

```bash
go work use ./iot-gateway-sdk ./iot-gateway-sdk/testkit ./iot-gateway-drivers
```
