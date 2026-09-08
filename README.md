# clankerbanker.com

An open-source web app for turning wallet and other custom CSV exports into Koinly-compatible reports. Phoenix Wallet and Xapo statements are supported; more converters can be added over time.

Conversion runs entirely in your browser using Go compiled to WebAssembly. Files are processed on your device, with no upload service or account required. The app can be built, inspected, and hosted locally. There is no CLI converter.

## Run locally

Install Go (the required version is in `go.mod`), Make, and Python 3, then run:

```sh
make serve
```

Open [localhost:8000](http://localhost:8000), choose Phoenix or Xapo, select or drop your export file(s), and download `koinly.csv` for import into Koinly. Stop the server with Ctrl+C.

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

## Xapo statements

Choose Xapo in the browser, then select or drop all Xapo statements for the export period together (for example, BTC account, USD account, and BTC interest). Download the consolidated `koinly.csv`. Keep Xapo's original filenames: they identify the account type when interpreting transfers.

Xapo's `Exchange USD to BTC` and `Exchange BTC to USD` activities appear in separate BTC and USD statement rows. When their transaction timestamp and sub-description match, the converter emits one trade: USD sent/BTC received or BTC sent/USD received, respectively. This keeps a BTC-funded card purchase as a BTC sale followed by a USD card payment, without double-counting either account leg. Card transaction rows are written as USD payments, while their card reference, merchant, and original GBP amount remain in the description.

A `Move to Savings` row is imported as a USD-to-BTC trade only when it came from a `USD_account` statement and has both an explicit BTC `Amount` and a `USD Amount`. The similarly shaped row in a `BTC_account` statement, and rows from an unrecognised filename, are omitted as internal or ambiguous transfers.

Daily BTC interest is marked `lending interest` without a fiat value, allowing Koinly to price it. For every non-BTC row, the converter uses Xapo's settled `USD Amount` as a USD movement; BTC rows use their explicit BTC `Amount`. The original Xapo action, counterparty, and sub-description—including any GBP or USDC reference—are retained in the Koinly description. Other received and outgoing account movements are mapped to deposits and withdrawals, and subscription fees to costs.

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

See [web/README.md](web/README.md) for static hosting instructions. The GitHub repository is [ExecutiveOrder6102/clankerbanker.com](https://github.com/ExecutiveOrder6102/clankerbanker.com). The Go module retains its original `phoenix-koinly-converter` path; the module path does not configure the website domain. See [OPENCODE.md](OPENCODE.md) for the contributor workflow.

## License

[BSD 3-Clause](LICENSE).
