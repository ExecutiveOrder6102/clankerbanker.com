# Web interface

clankerbanker.com is a static browser app backed by a Go WebAssembly converter. Phoenix Wallet and Xapo statements are supported CSV sources.

## Build and run

From the repository root, run `make serve` to build and serve the app at http://localhost:8000. This requires Go, Make, and Python 3. Alternatively run `make build` and serve `web/` using any static HTTP server. Opening `index.html` directly with a `file://` URL will not load the WebAssembly bundle reliably.

No backend, npm installation, or CLI converter is needed. Uploaded CSV content stays in browser memory, and the download is created locally.

## Convert CSVs

Choose Phoenix for a single wallet export, or Xapo to consolidate multiple BTC account, USD account, and interest statements. Select or drop the files, then download `koinly.csv`. For Xapo, select all statements for the export period together and keep their original filenames; account filenames affect how `Move to Savings` rows are interpreted. The rounding adjustment applies only to Phoenix.

See the root [README](../README.md) for each format's mapping and transfer handling.

## Static hosting

1. Run `make build` from the repository root.
2. Publish the contents of `web/`, including `main.wasm` and `wasm_exec.js`, to a static host.
3. Serve `.wasm` as `application/wasm` and JavaScript as `text/javascript` or `application/javascript`.
4. Deploy the Go runtime and WASM binary together. Revalidate these files on updates to avoid mismatched cached versions.

`_headers` supplies MIME, cache, and security headers for hosts that support that convention. Configure equivalent headers on other hosts. Hosts building from source should use `make build` as the build command and `web` as the publish directory.

Connect the clankerbanker.com domain through your hosting provider; the source code does not configure DNS.

## Browser checks

After changing the UI or WASM bridge, check that a Phoenix CSV can be selected and dropped, conversion produces a downloadable CSV, rounding can be toggled, and loading/conversion errors are visible. Check that Xapo accepts multiple files together and switching formats clears the previous selection and hides the Phoenix rounding option. `testdata/sample_phoenix.csv` is the synthetic Phoenix regression fixture; Xapo regression cases are in `converter/converter_test.go`.
