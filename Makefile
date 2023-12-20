.PHONY: all
all: build

.PHONY: build
build: clean
	goreleaser release --auto-snapshot --clean

.PHONY: publish
publish: clean
	goreleaser release --clean


.PHONY: clean
clean:
	go clean .
	-rm -rf vendor dist

.PHONY: vendor
vendor:
	go mod vendor

