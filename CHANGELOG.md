# Changelog

All notable changes to the iot-gateway SDK are documented here. This project
follows [Semantic Versioning](https://semver.org/). The wire protocol has its
own `APIVersion`; see the compatibility table in README.md.

## [0.1.0] - 2026-10-25

### Added

- Initial standalone SDK, extracted from the iot-gateway core.
- `driver`: driver contract, data types, quality codes, type registry,
  device/point discovery contracts.
- `pluginwire`: length-prefixed frame codec (`FrameMagic` = `IGP2`).
- `driversdk`: `Main`/`Serve`, plugin metadata, RPC parameter types, stderr
  logging.
- Apache-2.0 licensing, DCO contribution policy.
- Zero third-party dependencies; no import of the AGPL gateway core.

### Compatibility

- Wire `APIVersion`: 4.
- Gateway: v0.2.x and later, until the next protocol bump.
