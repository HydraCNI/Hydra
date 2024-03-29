
IMAGE_REGISTRY ?=docker.io/airren/
BUILD_VERSION ?= latest
#CNI_VERSION ?= v0.8.5

export IMAGE_NAME ?= $(IMAGE_REGISTRY)hydra-cni:$(BUILD_VERSION)

DOCKERARGS?=
ifdef HTTP_PROXY
	DOCKERARGS += --build-arg http_proxy=$(HTTP_PROXY)
endif
ifdef HTTPS_PROXY
	DOCKERARGS += --build-arg https_proxy=$(HTTPS_PROXY)
endif
#DOCKERARGS += --build-arg CNI_VERSION=$(CNI_VERSION)

build:
	go build -o ./bin/hydra-cni ./cmd/hydra
	go build -o ./bin/parallel-ep ./cmd/parallel_ep

build-img:
	podman build -t $(IMAGE_NAME) $(DOCKERARGS) --network host -f ./Dockerfile ./
	# podman build -t $(IMAGE_NAME) $(DOCKERARGS) --network host --no-cache -f ./Dockerfile ./

push:
	podman push $(IMAGE_NAME)

.PHONY: build_linux
build_linux:
	GOOS=linux GOARCH=amd64 go build -o ./bin/hydra-cni ./cmd



# sudo ctr -n=k8s.io i import ./hack/hydra-cni.tar
