FROM ubuntu:18.04

COPY bin/nfsshare-controller /usr/local/bin/

# We share in the kubeconfig here
VOLUME ["/kubeconfig"]

ENTRYPOINT ["/usr/local/bin/nfsshare-controller"]

# CMD  ["-kubeconfig", "/kubeconfig/config", "-v", "5", "-logtostderr"]
