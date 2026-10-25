# Hello driver

The smallest possible iot-gateway driver. It declares one protocol
(`Hello.TCP`) and returns `42` for every point.

```bash
go build -o plugins/iot-driver-hello .
```

Copy the binary into the gateway's `plugins/` directory, then create a device
with protocol `Hello.TCP`. This example is only for wiring up a build; replace
`Read` with your real protocol implementation.

For a real driver, keep the same shape and add:

- protocol-specific configuration parsing in `Connect`;
- reconnect and timeout handling;
- per-point quality codes (`driver.Quality*`) instead of always `QualityGood`;
- tests with a fake device (no PLC required in CI).
