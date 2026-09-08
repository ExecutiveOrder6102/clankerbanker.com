# Agent Guidelines

This repository is clankerbanker.com, a web-only app for converting CSV exports into Koinly reports. The conversion engine is Go compiled to WebAssembly; the UI is plain HTML, CSS, and JavaScript. Phoenix Wallet and Xapo statements are currently supported source formats.

## Coding Guidelines
- Format Go files using `gofmt -w` before committing.
- Keep functions small and focused, with straightforward loops and clear variable names.
- Reuse the existing conversion structs and helpers when extending functionality.
- Keep conversion entirely client-side and retain the ability to run locally.
- Update `web/README.md` when changing web behavior or build/serve instructions.

## Multi-Agent Guidance
- Use the explore agent to scan the codebase for entry points, file structure, and existing patterns.
- Use the general agent to implement changes and update tests/docs.
- Optional: run a second pass with an explore agent to review for missed references or doc updates.

## OpenCode Checklist
- Confirm the repo map and entry points before editing.
- Keep the converter and WASM browser bindings consistent when changing conversion logic.
- Update README when build steps or UI behavior change.
- Run `gofmt -w` on touched Go files.
- Run `go test ./...` before committing.

## Testing Guidelines
- Run `go test ./...` and `make build` before committing. All checks must pass.
- Preserve regression coverage when adding or changing CSV formats.
- If adding Go dependencies, run `go mod tidy`.

## Repository Structure
- `converter/` contains conversion logic and Go unit tests.
- `cmd/wasm/main.go` exposes the converter to the browser.
- `web/` contains the static browser interface.
- `testdata/` contains synthetic CSV fixtures.
