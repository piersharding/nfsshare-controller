FROM ubuntu:18.04

COPY bin/nfsshare-controller /usr/local/bin/

ENTRYPOINT ["nfsshare-controller"]
