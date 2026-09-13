GO=go
BUF=buf
SOURCE_TOPIC=kinetiq-test-topic
DEST_TOPIC=kinetiq-test-topic-out
ENV=
CHANGE_QUEUE=https://sqs.us-east-1.amazonaws.com/916325820950/kinetiq-updates-sqs
MOD_ROOT=test_module
MODULE=$(MOD_ROOT).wasm
MODULE_SRC=$(MOD_ROOT).go
BUCKET=kinetiq-test-bucket
DOCKER=docker
KAFKA_TOPICS=kafka-topics

gen-proto:
	-rm -rf gen
	$(BUF) generate

build: gen-proto
	$(GO) build

build-test-module:
	GOOS=wasip1 GOARCH=wasm $(GO) build -buildmode=c-shared -o examples/$(MOD_ROOT)/$(MODULE) examples/$(MODULE_SRC)

# build-test-module already writes to the path the running process watches, so
# the rebuild itself is what triggers the filesystem reload. The previous `mv`
# moved the file onto itself and did nothing.
hotswap-module-local: build-test-module
	@echo "Rebuilt $(MODULE) in place -- the running process should reload it."

hotswap-s3: build-test-module
	aws s3 cp examples/$(MOD_ROOT)/test_module.wasm s3://$(BUCKET)/test_module.wasm

run-test-module: build build-test-module
	if [ -z "$(SOURCE_TOPIC)" ] || [ -z "$(DEST_TOPIC)"]; then \
		echo "Error: SOURCE_TOPIC and DEST_TOPIC must be populated."; \
		exit 1; \
	fi
	$(ENV) $(GO) run main.go

create-topics:
	kafka-topics --bootstrap-server localhost:49092 --create --topic kinetiq-test-topic
	kafka-topics --bootstrap-server localhost:49092 --create --topic kinetiq-test-topic-out

start-kafka:
	$(DOCKER) compose up -d

stop-kafka:
	$(DOCKER) compose down

run-test-module-local:
	make run-test-module ENV="PLUGIN_REF=./examples/$(MOD_ROOT)/$(MODULE) KAFKA_SOURCE_TOPIC=$(SOURCE_TOPIC) KAFKA_DEST_TOPIC=$(DEST_TOPIC)"

run-test-module-s3:
	make run-test-module ENV="S3_INTEGRATION_ENABLED=true PLUGIN_REF=$(MODULE) KAFKA_SOURCE_TOPIC=$(SOURCE_TOPIC) KAFKA_DEST_TOPIC=$(DEST_TOPIC) S3_INTEGRATION_BUCKET=$(BUCKET) S3_INTEGRATION_CHANGE_QUEUE=$(CHANGE_QUEUE)"