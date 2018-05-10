.PHONY: build
build: target/initializer-container

SRCS=$(shell find ./pkg -name "*.go") $(shell find ./cmd -name "*.go")

.PHONY: generate
generate:
	operator-sdk generate k8s

.PHONY: containers
containers: target/initializer-container operator-container

.PHONY: push-container
push-containers: containers
	docker push soloio/envoy-operator:v0.0.1
	docker push soloio/envoy-operator-init:0.1

.PHONY: minikube-env
minikube-env:
	minikube start --vm-driver=kvm2 --cpus 3 --memory 8192
	minikube docker-env

.PHONY: deploy
deploy:
	kubectl apply -f deploy/rbac.yaml
	kubectl apply -f deploy/crd.yaml
	kubectl apply -f deploy/operator.yaml



################################## low level #############################################

target:
	[ -d $@ ] || mkdir -p $@

target/initializer: target $(SRCS)
	CGO_ENABLED=0 GOOS=linux go build -o $@ ./cmd/initializer/main.go

.PHONY: target/initializer-container
target/initializer-container: target/initializer cmd/initializer/Dockerfile
	cat cmd/initializer/Dockerfile | docker build -t soloio/envoy-operator-init:0.1 -f - target

.PHONY: operator-container
operator-container:
	operator-sdk build soloio/envoy-operator:v0.0.1