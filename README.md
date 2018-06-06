# nfsshare-controller

This repository implements a simple controller for watching Nfsshare resources as
defined with a CustomResourceDefinition (CRD).

This particular example demonstrates how to perform basic operations such as:

* How to register a new custom resource (custom resource type) of type `Nfsshare` using a CustomResourceDefinition.
* How to create/get/list instances of your new resource type `Nfsshare`.
* How to setup a controller on resource handling create/update/delete events.

This is an experiment only.

It makes use of the generators in [k8s.io/code-generator](https://github.com/kubernetes/code-generator)
to generate a typed client, informers, listers and deep-copy functions. You can
do this yourself using the `./hack/update-codegen.sh` script.

The `update-codegen` script will automatically generate the following files &
directories:

* `pkg/apis/nfssharecontroller/v1alpha1/zz_generated.deepcopy.go`
* `pkg/client/`

Changes should not be made to these files manually, and when creating your own
controller based off of this implementation you should not copy these files and
instead run the `update-codegen` script to generate your own.

## Build and Running

The nfsshare-operator need to be built, wrapped in a container, and then deployed as a single instance Pod on your cluster.
You can modify Makefile rules by creating your own PrivateRules.mak file - mine looks like:
```
REPO_PREFIX = ""
DOCKER_IMAGE = $(REPO_PREFIX)$(DEVICE):$(TAG)
```
Now work through:
```sh
# update the vendor dependencies if you haven't done so already:
$ make deps

# build the operator
$ make build

# Wrap the operator in a container
$ make docker

# Deploy to the currently configured cluster
$ make k8s_deploy
```

## Use
In the deploy/ directory there is an example of creating an nfsshare object `deploy/example-nfsshare.yaml` and then one for mounting the nfsshare into a Pod `deploy/nfstest.yaml`.  You will need to edit nfstest.yaml so that the volumes.nfs.server value reflects the ClusterIP of the nfsshare object.
You can find the nfsshare ClusterIP with:
```sh
$ kubectl get svc -o wide -l app=example-nfsshare
```

See the Makefile for other helpers.

