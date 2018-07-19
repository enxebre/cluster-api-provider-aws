all: build images

# Define constants
##################
BINDIR            ?= bin
PLATFORM          ?= linux
ARCH              ?= amd64
CLUSTERAPI_BIN     = $(BINDIR)/cluster-api
CLUSTERAPI_VERSION = $(shell git ls-remote https://github.com/kubernetes-sigs/cluster-api.git master | cut -c1-7)
VERSION           ?= $(shell git describe --always --abbrev=7)
GO_VERSION        ?= 1.10
GO_BUILD           = env GOOS=$(PLATFORM) GOARCH=$(ARCH) go build -i $(GOFLAGS)

AWS_MACHINE_CONTROLLER_PKG = github.com/enxebre/cluster-api-provider-aws

DOCKER_CMD = docker run --security-opt label:disable --rm -v $(PWD):/go/src/$(AWS_MACHINE_CONTROLLER_PKG) -v $(PWD)/.pkg:/go/pkg buildimage

MUTABLE_TAG                         ?= canary
CLUSTER_API_IMAGE                    = $(REGISTRY)cluster-api:$(CLUSTERAPI_VERSION)
CLUSTER_API_MUTABLE_IMAGE            = $(REGISTRY)cluster-api:$(MUTABLE_TAG)
CONTROLLER_MANAGER_IMAGE             = $(REGISTRY)controller-manager:$(CLUSTERAPI_VERSION)
CONTROLLER_MANAGER_MUTABLE_IMAGE     = $(REGISTRY)controller-manager:$(MUTABLE_TAG)
AWS_MACHINE_CONTROLLER_IMAGE         = $(REGISTRY)aws-machine-controller:$(VERSION)
AWS_MACHINE_CONTROLLER_MUTABLE_IMAGE = $(REGISTRY)aws-machine-controller:$(MUTABLE_TAG)

generate: gendeepcopy

#	go get -v k8s.io/code-generator/cmd/deepcopy-gen
#	deepcopy-gen \

gendeepcopy:
	/Users/albertogarla/go/src/k8s.io/code-generator/cmd/deepcopy-gen/deepcopy-gen \
	  -i ./awsproviderconfig,./awsproviderconfig/v1alpha1 \
	  -O zz_generated.deepcopy \
	  -h boilerplate.go.txt

# Some prereq stuff
###################

.buildImage: build/build-image/Dockerfile
	sed "s/GO_VERSION/$(GO_VERSION)/g" < build/build-image/Dockerfile | \
	  docker build -t buildimage -

.apiServerBuilderImage: build/apiserver-builder/Dockerfile
	sed "s/GO_VERSION/$(GO_VERSION)/g" < build/apiserver-builder/Dockerfile | \
	  docker build -t apiserverbuilderimage -

clean: clean-bin clean-images ## Clean everything

clean-bin: ## Remove build directory
	$(DOCKER_CMD) rm -rf $(BINDIR)

clean-images: ## Remove built images
	$(DOCKER_CMD) rm -rf .pkg
	docker rmi -f apiserverbuilderimage > /dev/null 2>&1 || true
	docker rmi -f buildimage > /dev/null 2>&1 || true

.PHONY: deps
deps: .buildImage
	$(DOCKER_CMD) glide install --strip-vendor
	$(DOCKER_CMD) glide-vc --use-lock-file --no-tests --only-code

# Build
#######

build: .buildImage apiserver aws-machine-controller ## Build all binaries
images: aws-machine-controller-image k8s-cluster-api-image k8s-controller-manager-image ## Create all images

.PHONY: $(BINDIR)/aws-machine-controller
aws-machine-controller: $(BINDIR)/aws-machine-controller ## Build aws-machine-controller binary
$(BINDIR)/aws-machine-controller: .buildImage
	mkdir -p $(PWD)/$(BINDIR)
	$(DOCKER_CMD) $(GO_BUILD) -o $@ $(AWS_MACHINE_CONTROLLER_PKG)/cmd

.PHONY: aws-machine-controller-image
aws-machine-controller-image: $(BINDIR)/aws-machine-controller aws-machine-controller ## Create aws-machine-controller image
	cp build/aws-machine-controller/Dockerfile $(BINDIR)/Dockerfile
	docker build -t $(AWS_MACHINE_CONTROLLER_IMAGE) ./$(BINDIR)
	docker tag $(AWS_MACHINE_CONTROLLER_IMAGE) $(AWS_MACHINE_CONTROLLER_MUTABLE_IMAGE)

# We need this to build the binary locally so docker volume works with buildimage
# then we build the image hitting the minikube docker daemon
.PHONY: aws-machine-controller-image-nobuild
aws-machine-controller-image-nobuild:
	cp build/aws-machine-controller/Dockerfile $(BINDIR)/Dockerfile
	docker build -t $(AWS_MACHINE_CONTROLLER_IMAGE) ./$(BINDIR)
	docker tag $(AWS_MACHINE_CONTROLLER_IMAGE) $(AWS_MACHINE_CONTROLLER_MUTABLE_IMAGE)

.PHONY: $(CLUSTERAPI_BIN)/apiserver
apiserver: $(CLUSTERAPI_BIN)/apiserver ## Build cluster-api and controller-manager binaries
$(CLUSTERAPI_BIN)/apiserver: .apiServerBuilderImage
	mkdir -p $(PWD)/$(CLUSTERAPI_BIN) && docker run --security-opt label:disable -v $(PWD)/$(CLUSTERAPI_BIN):/output --entrypoint=/bin/bash apiserverbuilderimage -c "export GOPATH=/go && mkdir -p /go/src/sigs.k8s.io/cluster-api && cd /go/src/sigs.k8s.io/cluster-api && git clone https://github.com/kubernetes-sigs/cluster-api.git . && apiserver-boot build executables --generate=false && touch /output/controller-manager /output/apiserver && cp bin/* /output"

.PHONY: k8s-cluster-api-image
k8s-cluster-api-image: $(CLUSTERAPI_BIN)/apiserver build/clusterapi-image/Dockerfile ## Build cluster-api image
	cp build/clusterapi-image/Dockerfile $(CLUSTERAPI_BIN)
	docker build -t $(CLUSTER_API_IMAGE) ./$(CLUSTERAPI_BIN)
	docker tag $(CLUSTER_API_IMAGE) $(CLUSTER_API_MUTABLE_IMAGE)

.PHONY: k8s-controller-manager-image
k8s-controller-manager-image: $(CLUSTERAPI_BIN)/apiserver build/controller-manager-image/Dockerfile ## Build controller-manager image
	cp build/controller-manager-image/Dockerfile $(CLUSTERAPI_BIN)
	docker build -t $(CONTROLLER_MANAGER_IMAGE) ./$(CLUSTERAPI_BIN)
	docker tag $(CONTROLLER_MANAGER_IMAGE) $(CONTROLLER_MANAGER_MUTABLE_IMAGE)

push: k8s-cluster-api-image k8s-controller-manager-image aws-machine-controller-image ## Push all images to registry
	docker push $(CLUSTER_API_IMAGE)
	docker push $(CLUSTER_API_MUTABLE_IMAGE)
	docker push $(CONTROLLER_MANAGER_IMAGE)
	docker push $(CONTROLLER_MANAGER_MUTABLE_IMAGE)
	docker push $(AWS_MACHINE_CONTROLLER_IMAGE)
	docker push $(AWS_MACHINE_CONTROLLER_MUTABLE_IMAGE)

.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
