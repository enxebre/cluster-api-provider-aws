BINDIR?= bin
PLATFORM?=linux
ARCH?=amd64

GO_BUILD= env GOOS=$(PLATFORM) GOARCH=$(ARCH) go build -i $(GOFLAGS) \

.PHONY: $(BINDIR)/aws-machine-controller
aws-machine-controller: $(BINDIR)/aws-machine-controller
$(BINDIR)/aws-machine-controller:
	$(GO_BUILD) -o $@ ./cmd

aws-machine-controller-image: $(BINDIR)/aws-machine-controller
	docker build -t aws-machine-controller --file build/aws-machine-controller/Dockerfile .
