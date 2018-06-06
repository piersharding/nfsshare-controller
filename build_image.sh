#/bin/sh

IMG=image_builder
WORKDIR=/go/src/github.com/piersharding/nfsshare-controller/
BIN=nfsshare-controller

docker rm -f ${IMG} || true
cat <<EOF >Dockerfile.builder
FROM golang:1.10
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN mkdir -p /results ${WORKDIR}
COPY . ${WORKDIR}
WORKDIR ${WORKDIR}
CMD /usr/bin/make deps  && /usr/bin/make build && cp bin/* /results/

EOF
docker build -t ${IMG} -f Dockerfile.builder .

# /builds/piersharding/nfsshare-controller
docker run --name gobuilder -it ${IMG}
docker cp gobuilder:/results/${BIN} bin/
echo "Output:"
ls -latr bin/
