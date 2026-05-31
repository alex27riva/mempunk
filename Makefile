BINARY  := mempunk
CMD     := ./cmd/mempunk
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test vet clean install

build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

install: build
	sudo cp $(BINARY) /home/mempunk/$(BINARY)
	sudo chown mempunk:mempunk /home/mempunk/$(BINARY)
