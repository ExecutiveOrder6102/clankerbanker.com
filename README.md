# Phoenix Koinly Converter

This project converts transaction data exported from the Phoenix Bitcoin Lightning wallet into a CSV format compatible with Koinly, cryptocurrency tax software. It includes both a traditional command-line workflow and an in-browser experience powered by WebAssembly.

## Features

- Converts Phoenix transaction types (lightning_received, lightning_sent, swap_in, swap_out, channel_open, channel_close) into Koinly-compatible records.
- Consolidates Xapo BTC, USD, and BTC-interest statements into one Koinly CSV.
- Handles amount conversions from millisats to BTC.
- CLI mode for converting CSV exports on your machine.
- In-browser converter (WASM) that runs entirely client-side with drag-and-drop upload and a downloadable `koinly.csv` output.
- Verbose logging flag for additional insight when debugging CLI runs.

## OpenCode workflow

If you are using OpenCode, this repo works best with a quick orientation pass followed by focused edits.

See `OPENCODE.md` for a one-page quickstart.

**Repo map**

- `main.go`: CLI entry point and conversion logic.
- `converter/`: parsing and conversion helpers.
- `cmd/wasm/`: WebAssembly entry point.
- `web/`: static UI and built WASM assets.

**Task recipes**

```bash
go run main.go <path_to_phoenix_csv_file>
make build-cli
make build-wasm
```

**Multi-agent prompts (examples)**

- "Explore conversion logic and list entry points for adding a new Phoenix transaction type."
- "Find where WASM bundles are built and update docs if the build steps change."
- "Add a new transaction type with tests; keep changes in converter/ and main.go consistent."

## Usage

### Command-line converter

Provide the path to your Phoenix CSV export file as a command-line argument. The converter writes a `koinly.csv` file to the current working directory.

```bash
go run main.go <path_to_phoenix_csv_file>
```

**Verbose mode example:**

```bash
go run main.go -v phoenix_transactions.csv
```

This generates `koinly.csv`, which you can import into Koinly.

### Xapo statements

Use `-xapo` followed by every Xapo statement in the export period (for example, BTC account, USD account, and BTC interest). The converter writes one consolidated `koinly.csv`.

```bash
go run main.go -xapo BTC_account.csv USD_account.csv BTC_interest.csv
```

Xapo's `Exchange USD to BTC` and `Exchange BTC to USD` activities are represented in separate BTC and USD statement rows. When their transaction timestamp and sub-description match, the converter emits one trade: USD sent/BTC received or BTC sent/USD received, respectively. This keeps a BTC-funded card purchase as a BTC sale followed by a USD card payment, rather than double-counting either account leg. Card transaction rows are written as USD payments, while their card reference, merchant, and original GBP amount remain in the description. A `Move to Savings` row is imported as a USD-to-BTC trade only when it came from a `USD_account` statement and has both an explicit BTC `Amount` and a `USD Amount`. The similarly shaped row in a `BTC_account` statement, and rows from an unrecognised filename, are omitted as internal or ambiguous transfers. Keep Xapo's original filenames when importing. Daily BTC interest is marked `lending interest` without a fiat value, allowing Koinly to price it. For every non-BTC row, the converter uses Xapo's settled `USD Amount` as a USD movement; BTC rows use their explicit BTC `Amount`. The original Xapo action, counterparty, and sub-description—including any GBP or USDC reference—are retained in the Koinly description. Other received and outgoing account movements are mapped to deposits and withdrawals, and subscription fees to costs.

### In-browser converter (WASM)

The `web/` directory contains a drag-and-drop interface that converts Phoenix exports entirely in your browser. Build the WebAssembly bundle and serve the static files locally:

```bash
make build-wasm
cd web
python -m http.server 8000
```

Open http://localhost:8000 in your browser, choose Phoenix or Xapo, then drop your export(s) and download the generated `koinly.csv` file. Choose Xapo to select multiple account and interest CSVs for a consolidated import. No data leaves your device.

### Verbose Logging

To enable verbose logging for detailed debugging output, use the `-v` flag:

```bash
go run main.go -v <path_to_phoenix_csv_file>
```

## Building from Source

Use the provided `Makefile` to build the CLI binary and the WASM bundle.

```bash
make build-cli   # Builds the CLI binary at ./phoenix-koinly-converter
make build-wasm  # Produces web/main.wasm and copies wasm_exec.js into web/
```

Then you can run the executable:

```bash
./phoenix-koinly-converter <path_to_phoenix_csv_file>
```

## Supported Phoenix Transaction Types

- `lightning_received`: Treated as received BTC.
- `lightning_sent`: Treated as sent BTC.
- `swap_in` / `legacy_swap_in`: Treated as received BTC (transfers).
- `swap_out`: Treated as sent BTC (transfers).
- `channel_open` / `legacy_pay_to_open`: Treated as received BTC (deposits).
- `channel_close`: Treated as a fee/cost in BTC.

Any other transaction types will be logged as unknown and may not be fully converted.
