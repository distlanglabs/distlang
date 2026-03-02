APP := distlang
CMD := ./cmd/distlang

.PHONY: help run build test fmt tidy clean

help:
	@echo "Targets:"
	@echo "  make run    - run distlang locally"
	@echo "  make build  - build binary into ./bin"
	@echo "  make test   - run tests"
	@echo "  make fmt    - format Go code"
	@echo "  make tidy   - tidy go modules"
	@echo "  make clean  - remove build artifacts"

run:
	go run $(CMD)

build:
	mkdir -p bin
	go build -o bin/$(APP) $(CMD)

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

clean:
	rm -rf bin
