
DEVICE:=nfsshare-controller
TAG ?= latest
REPO_PREFIX ?= gitlab.catalyst.net.nz:4567/piers/logging-and-monitoring/

.DEFAULT: $(DEVICE)

$(DEVICE):
	go build -i -o $(DEVICE) .

build: clean $(DEVICE)

buildall: clean $(DEVICE) bin/$(DEVICE)

all: clean deps docker

deps:
	dep ensure

clean:
	rm -f $(DEVICE) bin/$(DEVICE)

bin/$(DEVICE):
	./build

docker: build bin/$(DEVICE)
	docker build -t $(DEVICE):$(TAG) .

run:
	./bin/$(DEVICE) -h || true

docker_run:
	docker run --rm -ti $(DEVICE):$(TAG) -h || true

push:
	docker tag $(DEVICE):$(TAG) $(REPO_PREFIX)$(DEVICE):$(TAG)
	docker push $(REPO_PREFIX)$(DEVICE):$(TAG)
