.PHONY: build build-alpine clean test help default

BIN_NAME = sugarkube
BINDIR := $(CURDIR)/bin

VERSION ?= master
GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_DIRTY = $(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
BUILD_DATE = $(shell date '+%Y-%m-%d-%H:%M:%S')
IMAGE_NAME := "boosh/sugarkube"

default: test

help:
	@echo 'Management commands for sugarkube:'
	@echo
	@echo 'Usage:'
	@echo '    make build           Compile the project.'
	@echo '    make get-deps        runs dep ensure, mostly used for ci.'
	@echo '    make build-alpine    Compile optimized for alpine linux.'
	@echo '    make package         Build final docker image with just the go binary inside'
	@echo '    make tag             Tag image created by package with latest, git commit and version'
	@echo '    make test            Run tests on a compiled project.'
	@echo '    make push            Push tagged images to registry'
	@echo '    make clean           Clean the directory tree.'
	@echo

fmt:
	go fmt ./...

build: fmt
	@echo "building ${BIN_NAME} version=${VERSION}"
	@echo "GOPATH=${GOPATH}"
	@echo "GOBIN=${BINDIR}"
	GOBIN=$(BINDIR) go install -ldflags \
		"-X github.com/sugarkube/sugarkube/internal/pkg/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY} \
		 -X github.com/sugarkube/sugarkube/internal/pkg/version.BuildDate=${BUILD_DATE} \
		 -X github.com/sugarkube/sugarkube/internal/pkg/version.Version=${VERSION} \
		 " ./...

get-deps:
	dep ensure

build-alpine:
	@echo "building ${BIN_NAME} ${VERSION}"
	@echo "GOPATH=${GOPATH}"
	GOBIN=$(BINDIR) go install -ldflags '-w -linkmode external -extldflags "-static" -X github.com/sugarkube/sugarkube/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY} -X github.com/sugarkube/sugarkube/version.BuildDate=${BUILD_DATE}' ./...

package:
	@echo "building image ${BIN_NAME} ${VERSION} $(GIT_COMMIT)"
	docker build --build-arg VERSION=${VERSION} --build-arg GIT_COMMIT=$(GIT_COMMIT) -t $(IMAGE_NAME):local .

tag: 
	@echo "Tagging: latest ${VERSION} $(GIT_COMMIT)"
	docker tag $(IMAGE_NAME):local $(IMAGE_NAME):$(GIT_COMMIT)
	docker tag $(IMAGE_NAME):local $(IMAGE_NAME):${VERSION}
	docker tag $(IMAGE_NAME):local $(IMAGE_NAME):latest

push: tag
	@echo "Pushing docker image to registry: latest ${VERSION} $(GIT_COMMIT)"
	docker push $(IMAGE_NAME):$(GIT_COMMIT)
	docker push $(IMAGE_NAME):${VERSION}
	docker push $(IMAGE_NAME):latest

clean:
	-rm -rf bin/

test:
	go test ./...
