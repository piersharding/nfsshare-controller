#/bin/sh

IMG=image_builder
WORKDIR=/go/src/github.com/piersharding/nfsshare-controller/
PWD=`pwd`
docker rm -f ${IMG} || true
cat <<EOF >Dockerfile.builder
FROM golang:1.10
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN mkdir -p /src ${WORKDIR}
COPY . ${WORKDIR}
WORKDIR ${WORKDIR}
VOLUME ["/done"]
CMD /usr/bin/make deps  && /usr/bin/make build && cp bin/* /done/

EOF
docker build -t ${IMG} -f Dockerfile.builder .

#RUN ln -s ${WORKDIR} /src
#COPY . ${WORKDIR}
docker run --rm -it --mount "type=bind,src=${PWD}/bin,dst=/done" ${IMG}
echo "Output:"
ls -latr bin/

#docker run -it ${IMG} bash
#docker run --rm -it --mount "type=bind,src=${PWD}/bin,dst=/done" ${IMG} bash

