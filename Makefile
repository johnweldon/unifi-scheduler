NAME=unifi-scheduler
IMAGE=docker.w.jw4.us/$(NAME)
#PLATFORMS="linux/amd64,linux/arm64,linux/arm/v7"
PLATFORMS="linux/amd64,linux/arm64"

ifeq ($(BUILD_VERSION),)
	BUILD_VERSION := $(shell git describe --dirty --first-parent --always --tags)
endif

.PHONY: all
all: build

.PHONY: build
build: clean $(NAME)

.PHONY: clean
clean:
	go clean .
	-rm -rf vendor

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: push
push: vendor
	docker buildx build \
		--build-arg GOPROXY \
		--build-arg BUILD_VERSION=$(BUILD_VERSION) \
		-t $(IMAGE):$(BUILD_VERSION) \
		-t $(IMAGE):latest \
		--platform $(PLATFORMS) \
		--push \
		.

$(NAME):
	go build \
		-tags=netgo \
		-ldflags '-s -w -extldflags "-static"' \
		-ldflags "-X main.version=$(BUILD_VERSION)" \
		-o $(NAME) .

