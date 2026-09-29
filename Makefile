GO ?= go

.PHONY: build test check clean
build:
	mkdir -p build
	$(GO) build -trimpath -buildvcs=false -o build/quatrrod ./cmd/quatrrod
	$(GO) build -trimpath -buildvcs=false -o build/quatrroctl ./cmd/quatrroctl
	$(GO) build -trimpath -buildvcs=false -o build/quatrro-broker ./cmd/quatrro-broker
test:
	$(GO) test -race ./...
check:
	$(GO) vet ./...
	omarchy plugin validate .
clean:
	rm -rf build
