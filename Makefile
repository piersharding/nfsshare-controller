
DEVICE:=nfsshare-controller
TAG ?= latest
#REPO_PREFIX ?= http://localhost:5000/
REPO_PREFIX ?= registry.gitlab.com/piersharding/nfsshare-controller/
DOCKER_IMAGE = $(REPO_PREFIX)$(DEVICE):$(TAG)
CURRENT_DIR = $(shell pwd)
IMG_BUILDER ?= image_builder
CONTROLLER_BIN = nfsshare-controller
WORKDIR = ${GOPATH}/src/github.com/piersharding/${CONTROLLER_BIN}/
# /builds/piersharding/nfsshare-controller
CI_PROJECT_DIR ?= ${CURRENT_DIR}
BUILD_PATH = ${CI_PROJECT_DIR}
GOLANG ?= golang:1.10
BUILDER_FILE = Dockerfile.builder


# define overides for above variables in here
-include PrivateRules.mak

checkvars:
	@echo "Image: $(DOCKER_IMAGE)"
	@echo "Repo: $(REPO_PREFIX)"
	@echo "Workdir: $(WORKDIR)"
	@echo "Golang: $(GOLANG)"
	@echo "Buildpath: $(BUILD_PATH)"

.DEFAULT: $(DEVICE)

$(DEVICE):
	go build -i -o $(DEVICE) .

comp: $(DEVICE)

build: clean bin/$(DEVICE)

all: clean deps docker

deps:
	dep ensure
	git checkout 

clean:
	@echo "Start with clean: " `date`
	rm -f $(DEVICE) bin/$(DEVICE)

bin/$(DEVICE):
	./build

docker: build 
	docker build -t $(DEVICE):$(TAG) .
	@echo "Finished image build: " `date`

run: build
	./bin/$(DEVICE) -h || true

docker_run: docker
	docker run --name nfs-server -v $(CURRENT_DIR)/kubeconfig:/kubeconfig -d $(DEVICE):$(TAG) -kubeconfig /kubeconfig/config -v 5 -logtostderr

docker_rm:
	docker rm -f nfs-server

run_builder:
	@echo "Started builder at: " `date`
	# /builds/piersharding/nfsshare-controller
	@echo "BUILD_PATH is: ${BUILD_PATH}"
	@echo "GOLANG base image version is: ${GOLANG}"
	@echo "WORKDIR is: ${WORKDIR}"
	@echo "IMG builder is: ${IMG_BUILDER}"
	@echo "Result binary is: ${CONTROLLER_BIN}"
	docker rm -f ${IMG_BUILDER} || true
	echo "FROM $(GOLANG)" >>$(BUILDER_FILE)
	echo "ENV GOPATH /go" >>$(BUILDER_FILE)
	echo "RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh" >>$(BUILDER_FILE)
	echo "RUN mkdir -p /results $(WORKDIR)" >>$(BUILDER_FILE)
	echo "COPY . $(WORKDIR)" >>$(BUILDER_FILE)
	echo "WORKDIR $(WORKDIR)" >>$(BUILDER_FILE)
	echo "CMD /usr/bin/make deps && /usr/bin/make build && ls -l bin/ && cp bin/* /results/" >>$(BUILDER_FILE)
	docker build -t $(IMG_BUILDER) -f $(BUILDER_FILE) .
	@echo "Build $(CONTROLLER_BIN) in docker"
	docker run --name $(IMG_BUILDER) \
	   -v $(BUILD_PATH)/bin:/results \
	   $(IMG_BUILDER)
	@echo "Finished build."
	@echo "Does the container still exist:"
	docker ps -a
	@echo "Output:"
	ls -latr $(BUILD_PATH)/bin/
	@echo "Finished builder at: " `date`

k8s_deploy: crd
	DOCKER_IMAGE=$(DOCKER_IMAGE) \
	 envsubst < deploy/nfsshare-operator-deployment.yaml | kubectl apply -f -

k8s_delete:
	DOCKER_IMAGE=$(DOCKER_IMAGE) \
	 envsubst < deploy/nfsshare-operator-deployment.yaml | kubectl delete -f - || true

push:
	docker tag $(DEVICE):$(TAG) $(DOCKER_IMAGE)
	docker push $(DOCKER_IMAGE)

crd: delete-crd
	kubectl apply -f deploy/crd.yaml
	kubectl get crd

delete-crd:
	kubectl delete crd nfsshares.skasdp.org || true

test:
	kubectl apply -f deploy/example-nfsshare.yaml

delete-test:
	kubectl delete -f deploy/example-nfsshare.yaml
