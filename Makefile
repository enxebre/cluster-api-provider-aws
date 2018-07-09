all: build

# Define constants
##################
BINDIR        ?= bin
PLATFORM      ?= linux
ARCH          ?= amd64
CLUSTERAPI_BIN = $(BINDIR)/cluster-api
VERSION       ?= $(shell git describe --always --abbrev=7 --dirty)
GO_VERSION     = 1.9
GO_BUILD       = env GOOS=$(PLATFORM) GOARCH=$(ARCH) go build -i $(GOFLAGS)

CLUSTER_API_IMAGE = $(REGISTRY)cluster-api:$(VERSION)
CONTROLLER_MANAGER_IMAGE = $(REGISTRY)controller-manager:$(VERSION)
AWS_MACHINE_CONTROLLER_IMAGE = $(REGISTRY)aws-machine-controller:$(VERSION)

# Some prereq stuff
###################

.apiServerBuilderImage: build/apiserver-builder/Dockerfile
	sed "s/GO_VERSION/$(GO_VERSION)/g" < build/apiserver-builder/Dockerfile | \
	  docker build -t apiserverbuilderimage -

clean: .clean-bin .clean-build-image

clean-bin:
	rm -rf $(BINDIR)

clean-build-image:
	docker rmi -f apiserverbuilderimage > /dev/null 2>&1 || true

# Build
#######

build: kubernetes-cluster-api kubernetes-controller-manager aws-machine-controller

.PHONY: $(BINDIR)/aws-machine-controller
aws-machine-controller: $(BINDIR)/aws-machine-controller
$(BINDIR)/aws-machine-controller:
  $(GO_BUILD) -o $@ ./cmd

aws-machine-controller-image: $(BINDIR)/aws-machine-controller
  docker build -t $(AWS_MACHINE_CONTROLLER_IMAGE) --file build/aws-machine-controller/Dockerfile .

.PHONY: $(CLUSTERAPI_BIN)/apiserver
$(CLUSTERAPI_BIN)/apiserver: .apiServerBuilderImage
	mkdir -p $(PWD)/$(CLUSTERAPI_BIN) && docker run --security-opt label:disable -v $(PWD)/$(CLUSTERAPI_BIN):/output --entrypoint=/bin/bash apiserverbuilderimage -c "export GOPATH=/go && mkdir -p /go/src/sigs.k8s.io/cluster-api && cd /go/src/sigs.k8s.io/cluster-api && git clone https://github.com/kubernetes-sigs/cluster-api.git . && apiserver-boot build executables --generate=false && touch /output/controller-manager /output/apiserver && cp bin/* /output"

.PHONY: kubernetes-cluster-api
kubernetes-cluster-api: $(CLUSTERAPI_BIN)/apiserver build/clusterapi-image/Dockerfile
	cp build/clusterapi-image/Dockerfile $(CLUSTERAPI_BIN)
	docker build -t $(CLUSTER_API_IMAGE) ./$(CLUSTERAPI_BIN)

kubernetes-controller-manager: $(CLUSTERAPI_BIN)/controller-manager build/controller-manager-image/Dockerfile
	cp build/controller-manager-image/Dockerfile $(CLUSTERAPI_BIN)
	docker build -t $(CONTROLLER_MANAGER_IMAGE) ./$(CLUSTERAPI_BIN)

push: kubernetes-cluster-api kubernetes-controller-manager aws-machine-controller-image
	docker push $(CLUSTER_API_IMAGE)
	docker push $(CONTROLLER_MANAGER_IMAGE)