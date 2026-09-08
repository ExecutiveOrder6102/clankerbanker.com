# Web interface

clankerbanker.com is a static browser app backed by a Go WebAssembly converter. Phoenix Wallet is currently the only supported CSV source.

## Build and run

From the repository root, run `make serve` to build and serve the app at http://localhost:8000. This requires Go, Make, and Python 3. Alternatively run `make build` and serve `web/` using any static HTTP server. Opening `index.html` directly with a `file://` URL will not load the WebAssembly bundle reliably.

No backend, npm installation, or CLI converter is needed. Uploaded CSV content stays in browser memory, and the download is created locally.

## Static hosting

1. Run `make build` from the repository root.
2. Publish the contents of `web/`, including `main.wasm` and `wasm_exec.js`, to a static host.
3. Serve `.wasm` as `application/wasm` and JavaScript as `text/javascript` or `application/javascript`.
4. Deploy the Go runtime and WASM binary together. Revalidate these files on updates to avoid mismatched cached versions.

`_headers` supplies MIME, cache, and security headers for hosts that support that convention. Configure equivalent headers on other hosts. Hosts building from source should use `make build` as the build command and `web` as the publish directory.

Connecting the clankerbanker.com domain and renaming the GitHub repository are separate hosting/account operations; the source code does not configure DNS.

## Browser checks

After changing the UI or WASM bridge, check that a sample CSV can be selected and dropped, conversion produces a downloadable CSV, rounding can be toggled, and loading/conversion errors are visible. `testdata/sample_phoenix.csv` is the synthetic regression fixture.
