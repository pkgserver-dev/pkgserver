VERSION ?= latest
REGISTRY ?= europe-docker.pkg.dev/srlinux/eu.gcr.io
PROJECT ?= pkg-server
IMG ?= $(REGISTRY)/${PROJECT}:$(VERSION)

REPO = github.com/pkgserver-dev/pkgserver

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.15.0
KFORM ?= $(LOCALBIN)/kform
KFORM_VERSION ?= v0.0.4

.PHONY: codegen fix fmt vet lint test tidy

GOBIN := $(shell go env GOPATH)/bin

all: codegen fmt vet lint test tidy

.PHONY:
build:
	mkdir -p bin
	CGO_ENABLED=0 ${GOBIN} build -o bin/pkgctl cmd/pkgctl/main.go 

.PHONY:
docker:
	GOOS=linux GOARCH=arm64 go build -o install/bin/apiserver
	docker build install --tag apiserver-caas:v0.0.0

.PHONY:
docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push:  docker-build ## Push docker image with the manager.
	docker push ${IMG}

install: docker
	kustomize build install | kubectl apply -f -

reinstall: docker
	kustomize build install | kubectl apply -f -
	kubectl delete pods -n config-system --all

apiserver-logs:
	kubectl logs -l apiserver=true --container apiserver -n config-system -f --tail 1000

codegen:
	(which apiserver-runtime-gen || go get sigs.k8s.io/apiserver-runtime/tools/apiserver-runtime-gen)
	go generate

genclients:
	go run ./tools/apiserver-runtime-gen \
		-g client-gen \
		-g deepcopy-gen \
		-g informer-gen \
		-g lister-gen \
		-g openapi-gen \
		-g go-to-protobuf \
		--module $(REPO) \
		--versions $(REPO)/apis/pkg/v1alpha1,$(REPO)/apis/config/v1alpha1,$(REPO)/apis/pkgid,$(REPO)/apis/condition

#		-g go-to-protobuf \

.PHONY: generate
generate: controller-gen 
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./apis/pkgid/..."
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./apis/condition/..."
	protoc -I . $(shell find ./pkg/ -name '*.proto') --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative


.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./apis/config/..." output:crd:artifacts:config=artifacts

.PHONY: artifacts
artifacts: kform
	mkdir -p artifacts/out
	if [ ! -e "artifacts/.kform/kform-inventory.yaml" ]; then \
        $(KFORM) init artifacts; \
    fi
	$(KFORM) apply artifacts -i artifacts/in/configmap-input-vars.yaml -o artifacts/out/artifacts.yaml

fix:
	go fix ./...

fmt:
	test -z $(go fmt ./tools/...)

tidy:
	go mod tidy

lint:
	(which golangci-lint || go get github.com/golangci/golangci-lint/cmd/golangci-lint)
	$(GOBIN)/golangci-lint run ./...

test:
	go test -cover ./...

vet:
	go vet ./...

local-run:
	apiserver-boot run local --run=etcd,apiserver

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: kform
kform: $(KFORM) ## Download kform locally if necessary.
$(KFORM): $(LOCALBIN)
	test -s $(LOCALBIN)/kform || GOBIN=$(LOCALBIN) go install github.com/kform-dev/kform/cmd/kform@$(KFORM_VERSION)