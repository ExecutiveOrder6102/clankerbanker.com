# Web interface

clankerbanker.com is a static browser app backed by a Go WebAssembly converter. Phoenix Wallet and Xapo statements are supported CSV sources.

## Build and run

From the repository root, run `make serve` to build and serve the app at http://localhost:8000. This requires Go, Make, and Python 3. Alternatively run `make build` and serve `web/` using any static HTTP server. Opening `index.html` directly with a `file://` URL will not load the WebAssembly bundle reliably.

No backend, npm installation, or CLI converter is needed. Uploaded CSV content stays in browser memory, and the download is created locally.

## Convert CSVs

1. Choose Phoenix for a single wallet export, or Xapo to consolidate multiple BTC account, USD account, and interest statements. Search matches source names, categories, and descriptions.
2. Select or drop CSV files. Review the selected filenames and remove any unwanted files. Adding more Xapo files keeps the current selection; adding a file with the same name replaces it. Keep original Xapo filenames because account filenames affect how `Move to Savings` rows are interpreted.
3. For Phoenix, optionally enable the rounding adjustment under **Conversion options**.
4. Click **Convert to Koinly**, then download `koinly.csv` when conversion completes.

Switching sources clears the selected files. Changing files or conversion options clears the previous download, so it always reflects the last completed conversion. All processing happens on your device; the pixel-art background is served with the app and does not require an external image service.

Source cards and source-specific instructions are defined in the `sources` catalogue in `app.js`. New formats can extend that catalogue with their file selection rules and converter binding. The Night Shift layout adapts to narrow screens and keeps the decorative city behind the conversion workspace.

See the root [README](../README.md) for each format's mapping and transfer handling.

## Static hosting

1. Run `make build` from the repository root.
2. Publish the contents of `web/`, including `assets/`, `main.wasm`, and `wasm_exec.js`, to a static host.
3. Serve `.wasm` as `application/wasm` and JavaScript as `text/javascript` or `application/javascript`.
4. Deploy the Go runtime and WASM binary together. Revalidate these files on updates to avoid mismatched cached versions.

`_headers` supplies MIME, cache, and security headers for hosts that support that convention. Configure equivalent headers on other hosts. Hosts building from source should use `make build` as the build command and `web` as the publish directory.

Connect the clankerbanker.com domain through your hosting provider; the source code does not configure DNS.

## Browser verification

Run `node --test tests/web.test.cjs` from the repository root for dependency-free UI state regression tests. These cover source selection, file validation, result invalidation, asynchronous file reads, and converter loading errors using a minimal DOM adapter and mocked conversion bindings. Run `go test ./...` and `make build` for the conversion engine and WASM build.

After changing the UI or WASM bridge, check that a Phoenix CSV can be selected and dropped, conversion produces a downloadable CSV, rounding can be toggled, and loading/conversion errors are visible. Check that Xapo accepts multiple files together and switching formats clears the previous selection and hides the Phoenix rounding option. `testdata/sample_phoenix.csv` is the synthetic Phoenix regression fixture; Xapo regression cases are in `converter/converter_test.go`.
