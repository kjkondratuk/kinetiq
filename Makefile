GO=go
BUF=buf

gen-proto:
	-rm -rf gen
	$(BUF) generate

build: gen-proto
	$(GO) build

build-test-module: gen-proto
	cd examples/module && \
	GOOS=wasip1 GOARCH=wasm $(GO) build -buildmode=c-shared -o test_module.wasm

run-test-module: build build-test-module
	PLUGIN_REF=./examples/module/test_module.wasm $(GO) run main.go