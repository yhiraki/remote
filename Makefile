# Get the version from the latest git tag
VERSION := $(shell git describe --tags --always)
# Set the linker flags to inject the version
LDFLAGS := -ldflags="-X main.version=${VERSION}"

.PHONY: build
build:
	@echo "==> Building remote version ${VERSION}..."
	@go build ${LDFLAGS} -o remote .

.PHONY: install
install:
	@go install ${LDFLAGS}

.PHONY: clean
clean:
	@rm -f remote
