# clankerbanker.com

An open-source web app for turning wallet and other custom CSV exports into Koinly-compatible reports. Phoenix Wallet is the first supported format; more converters can be added over time.

Conversion runs entirely in your browser using Go compiled to WebAssembly. Files are processed on your device, with no upload service or account required. The app can be built, inspected, and hosted locally. There is no CLI converter.

## Run locally

Install Go (the required version is in `go.mod`), Make, and Python 3, then run:

```sh
make serve
```

Open [localhost:8000](http://localhost:8000), choose or drop a Phoenix CSV export, and download `koinly.csv` for import into Koinly. Stop the server with Ctrl+C.

To build and serve separately:

```sh
make build
python3 -m http.server 8000 --bind 127.0.0.1 --directory web
```

The build creates `web/main.wasm` and copies the matching Go runtime into `web/wasm_exec.js`. Both are generated files. Rebuild after changes to Go code or the Go toolchain. No JavaScript package installation is required.

## Supported Phoenix transactions

| Phoenix type | Koinly mapping |
| --- | --- |
| `lightning_received` | Received BTC, `lightning` label |
| `lightning_sent` | Sent BTC, `lightning` label |
| `swap_in`, `legacy_swap_in` | Received BTC, `transfer` label |
| `swap_out` | Sent BTC, `transfer` label |
| `channel_open`, `legacy_pay_to_open` | Received BTC, `deposit` label |
| `channel_close` | BTC fee, `cost` label |

Amounts are converted from millisats to BTC with eight decimal places. The optional rounding adjustment adds a cost transaction when accumulated rounding differences reach a whole satoshi after rounding. This entry is dated at conversion time.

Existing conversion behavior is retained: invalid timestamps are skipped, invalid integer fields become zero, and unknown transaction types produce incomplete rows and a console warning. Review the resulting report before import. The converter expects the Phoenix column layout, not arbitrary CSVs.

## Development and testing

```sh
make test     # Go converter tests, including the original regression tests
make build    # Compile and validate the browser entry point
make clean    # Remove generated browser binaries/runtime
```

GitHub Actions runs the Go tests and WebAssembly build on pull requests and pushes to `main`.

- `converter/`: shared conversion logic and tests.
- `cmd/wasm/main.go`: browser bindings for the conversion engine.
- `web/`: static interface, styles, and generated browser bundle.
- `testdata/`: synthetic CSV fixtures.

To add another CSV format, implement a parser and Koinly mapping in Go, add fixture-based tests, expose it through the WebAssembly entry point, and add format selection to the web interface. Keep all file processing in the browser.

See [web/README.md](web/README.md) for static hosting instructions. The repository link and Go module path retain their current GitHub location until the repository itself is renamed; neither is the website's domain configuration.

## License

[BSD 3-Clause](LICENSE).
