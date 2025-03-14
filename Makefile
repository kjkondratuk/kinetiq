GO=go
BUF=buf
SOURCE_TOPIC=kinetiq-test-topic
DEST_TOPIC=kinetiq-test-topic-out
ENV=
CHANGE_QUEUE=https://sqs.us-east-1.amazonaws.com/916325820950/kinetiq-updates-sqs
MODULE=test_module.wasm
BUCKET=kinetiq-test-bucket

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
	$(ENV) $(GO) run main.go

start-kafka:
	docker-compose up -d

stop-kafka:
	docker-compose down

run-test-module-local:
	make run-test-module ENV="PLUGIN_REF=./examples/module/$(MODULE) KAFKA_SOURCE_TOPIC=$(SOURCE_TOPIC) KAFKA_DEST_TOPIC=$(DEST_TOPIC)"

run-test-module-s3:
	make run-test-module ENV="S3_INTEGRATION_ENABLED=true PLUGIN_REF=$(MODULE) KAFKA_SOURCE_TOPIC=$(SOURCE_TOPIC) KAFKA_DEST_TOPIC=$(DEST_TOPIC) S3_INTEGRATION_BUCKET=$(BUCKET) S3_INTEGRATION_CHANGE_QUEUE=$(CHANGE_QUEUE)"