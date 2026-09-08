# OpenCode Quickstart

This repo is clankerbanker.com, a web-only app for converting Phoenix Wallet and Xapo CSV exports into Koinly CSV output. Go conversion code runs entirely in the browser through WebAssembly and can be hosted locally.

## Repo map

- `converter/`: parsing, conversion helpers, and regression tests for Phoenix and Xapo.
- `cmd/wasm/`: WebAssembly entry point.
- `web/`: static UI and built WASM assets.
- `testdata/`: sample Phoenix CSVs used in tests.

## Task recipes

```bash
make test    # Run the Go regression tests
make build   # Build the static WASM bundle
make serve   # Build and serve locally at http://localhost:8000
```

## Multi-agent workflow

1. Explore agent: locate entry points, patterns, and any existing tests.
2. General agent: implement the change and update docs/tests.
3. Optional explore pass: verify documentation and references are updated.

## Example prompts

- "Explore how conversion logic handles Phoenix transaction types and list the files to update for new types."
- "Add a new transaction type conversion and update tests to cover it."
- "Update WASM build or UI behavior and ensure README instructions stay accurate."
