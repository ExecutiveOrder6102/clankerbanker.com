.PHONY: all build build-wasm serve test clean

all: build

build: build-wasm

build-wasm:
	@runtime="$$(go env GOROOT)/lib/wasm/wasm_exec.js"; \
	if [ ! -f "$$runtime" ]; then \
		runtime="$$(go env GOROOT)/misc/wasm/wasm_exec.js"; \
	fi; \
	cp "$$runtime" web/wasm_exec.js
	GOOS=js GOARCH=wasm go build -o web/main.wasm ./cmd/wasm

serve: build
	python3 -m http.server 8000 --bind 127.0.0.1 --directory web

test:
	go test ./...

clean:
	rm -f web/main.wasm web/wasm_exec.js
