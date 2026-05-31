BINARY  := mempunk
CMD     := ./cmd/mempunk
OUT     := build/$(BINARY)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test vet clean install

build:
	mkdir -p build
	go build $(LDFLAGS) -o $(OUT) $(CMD)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf build/

install: build
	sudo cp $(OUT) /home/mempunk/$(BINARY)
	sudo chown mempunk:mempunk /home/mempunk/$(BINARY)
