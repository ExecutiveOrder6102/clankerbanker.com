# Agent Guidelines

This repository is clankerbanker.com, a web-only app for converting CSV exports into Koinly reports. The conversion engine is Go compiled to WebAssembly; the UI is plain HTML, CSS, and JavaScript. Phoenix Wallet is currently the supported source format.

## Coding Guidelines
- Format Go files using `gofmt -w` before committing.
- Keep functions small and focused, with straightforward loops and clear variable names.
- Reuse the existing conversion structs and helpers when extending functionality.
- Keep conversion entirely client-side and retain the ability to run locally.
- Update `web/README.md` when changing web behavior or build/serve instructions.

## Testing Guidelines
- Run `go test ./...` and `make build` before committing. All checks must pass.
- Preserve regression coverage when adding or changing CSV formats.
- If adding Go dependencies, run `go mod tidy`.

## Repository Structure
- `converter/` contains conversion logic and Go unit tests.
- `cmd/wasm/main.go` exposes the converter to the browser.
- `web/` contains the static browser interface.
- `testdata/` contains synthetic CSV fixtures.
