# OpenCode Quickstart

This repo is a Go utility for converting Phoenix wallet CSV exports into Koinly CSV output. It supports both a CLI and a WebAssembly-powered browser UI.

## Repo map

- `main.go`: CLI entry point and conversion logic.
- `converter/`: parsing and conversion helpers.
- `cmd/wasm/`: WebAssembly entry point.
- `web/`: static UI and built WASM assets.
- `testdata/`: sample Phoenix CSVs used in tests.

## Task recipes

```bash
go run main.go <path_to_phoenix_csv_file>
make build-cli
make build-wasm
```

## Multi-agent workflow

1. Explore agent: locate entry points, patterns, and any existing tests.
2. General agent: implement the change and update docs/tests.
3. Optional explore pass: verify documentation and references are updated.

## Example prompts

- "Explore how conversion logic handles Phoenix transaction types and list the files to update for new types."
- "Add a new transaction type conversion and update tests to cover it."
- "Update WASM build or UI behavior and ensure README instructions stay accurate."
