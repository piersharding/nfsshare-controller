#/bin/sh

IMG=image_builder
BIN=nfsshare-controller
WORKDIR=/go/src/github.com/piersharding/${BIN}/
HERE=$(pwd)
BUILD_PATH=${CI_PROJECT_DIR-$HERE}
# /builds/piersharding/nfsshare-controller

echo "BUILD_PATH is: $BUILD_PATH"

docker rm -f ${IMG} || true
cat <<EOF >Dockerfile.builder
FROM golang:1.10
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN mkdir -p /results ${WORKDIR}
COPY . ${WORKDIR}
WORKDIR ${WORKDIR}
CMD /usr/bin/make deps  && /usr/bin/make build && ls -l bin/ && cp bin/* /results/
EOF
docker build -t ${IMG} -f Dockerfile.builder .

rm -f ${BUILD_PATH}/bin/*
echo "Build ${BIN} in docker"

docker run --name gobuilder \
   -v ${BUILD_PATH}/bin:/results \
   ${IMG}
echo "Finished build."
echo "Does the container still exist:"
docker ps -a
echo "Output:"
ls -latr ${BUILD_PATH}/bin/
