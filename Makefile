GO=go
BUF=buf
SOURCE_TOPIC=kinetiq-test-topic
DEST_TOPIC=kinetiq-test-topic-out

gen-proto:
	-rm -rf gen
	$(BUF) generate

build: gen-proto
	$(GO) build

build-test-module: gen-proto
	cd examples/module && \
	GOOS=wasip1 GOARCH=wasm $(GO) build -buildmode=c-shared -o test_module.wasm

run-test-module: build build-test-module
	if [ -z "$(SOURCE_TOPIC)" ] || [ -z "$(DEST_TOPIC)"]; then \
		echo "Error: SOURCE_TOPIC and DEST_TOPIC must be populated."; \
		exit 1; \
	fi
	PLUGIN_REF=./examples/module/test_module.wasm KAFKA_SOURCE_TOPIC=$(SOURCE_TOPIC) KAFKA_DEST_TOPIC=$(DEST_TOPIC) $(GO) run main.go

start-kafka:
	docker-compose up -d
	