GO=go
BUF=buf

gen-proto:
	-rm -rf gen
	$(BUF) generate

build:
	$(GO) build