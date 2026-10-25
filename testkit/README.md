# testkit — fake PLC servers for driver tests

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
