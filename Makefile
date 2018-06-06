
DEVICE:=nfsshare-controller
TAG ?= latest
REPO_PREFIX ?= http://localhost:5000/
DOCKER_IMAGE = $(REPO_PREFIX)$(DEVICE):$(TAG)
CURRENT_DIR = $(shell pwd)

# define overides for above variables in here
-include PrivateRules.mak

checkvars:
	@echo "Image: $(DOCKER_IMAGE)"
	@echo "Repo: $(REPO_PREFIX)"

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
	@date
	rm -f $(DEVICE) bin/$(DEVICE)

bin/$(DEVICE):
	./build

docker: build 
	docker build -t $(DEVICE):$(TAG) .

run: build
	./bin/$(DEVICE) -h || true

docker_run: docker
	docker run --name nfs-server -v $(CURRENT_DIR)/kubeconfig:/kubeconfig -d $(DEVICE):$(TAG) -kubeconfig /kubeconfig/config -v 5 -logtostderr

docker_rm:
	docker rm -f nfs-server

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
