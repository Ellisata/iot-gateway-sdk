# Contributing to the iot-gateway SDK

[简体中文](CONTRIBUTING.md) | English

## License of contributions

This SDK is licensed under **Apache-2.0**. By contributing, you agree that your
contribution is licensed under Apache-2.0 (inbound = outbound). **No CLA is
required.**

Every commit must carry a **Developer Certificate of Origin** sign-off:

```bash
git commit -s -m "driver: add ..."
```

The sign-off certifies the statements in [DCO](DCO). CI rejects pull requests
whose commits lack a matching `Signed-off-by:` trailer.

## Hard rules

1. **Never import the AGPL gateway core.** The SDK must remain usable by
   proprietary drivers. CI runs a dependency-direction check; a PR that adds an
   `iot-gateway` import will fail.
2. **No third-party runtime dependencies** without prior discussion. The SDK is
   deliberately stdlib-only so a driver has no supply-chain surface beyond the
   Go toolchain.
3. **Wire compatibility is a contract.** Do not change JSON field names, frame
   layout, or binary payload order without bumping `driversdk.APIVersion` and
   documenting the migration. Golden/parity tests must stay green.
4. **Apache-2.0 SPDX header** is required on every new file:

   ```go
   // SPDX-License-Identifier: Apache-2.0
   // Copyright (c) 2026 Edwin and iot-gateway contributors
   ```

## Development

```bash
go test ./...
gofmt -l .
go vet ./...
```

The test suite is stdlib-only and runs without a gateway or a PLC.

## Reporting security issues

Do not open a public issue for a vulnerability. Follow the security policy in
the main repository (SECURITY.md) or contact the maintainers privately.
