.PHONY: default
default: build

.PHONY: build
build:
	cd v2; go build -o bin/launcher

.PHONY: test
test:
	cd v2; go test ./...

.PHONY: release
release:
	export VERSION=$(shell cd v2; go run ./... --version) &&\
	git tag $$VERSION && git push refs/tags/$$VERSION
