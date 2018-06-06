#/bin/sh
echo "Started at: " `date`

IMG=image_builder
BIN=nfsshare-controller
WORKDIR=/go/src/github.com/piersharding/${BIN}/
# /builds/piersharding/nfsshare-controller
HERE=$(pwd)
BUILD_PATH=${CI_PROJECT_DIR-$HERE}
GOLANG=${GOLANG-golang:1.10}

echo "BUILD_PATH is: ${BUILD_PATH}"
echo "GOLANG base image version is: ${GOLANG}"
echo "WORKDIR is: ${WORKDIR}"
echo "IMG builder is: ${IMG}"
echo "Result binary is: ${BIN}"

docker rm -f ${IMG} || true
cat <<EOF >Dockerfile.builder
FROM ${GOLANG}
ENV GOPATH /go
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN mkdir -p /results ${WORKDIR}
COPY . ${WORKDIR}
WORKDIR ${WORKDIR}
CMD /usr/bin/make deps && /usr/bin/make build && ls -l bin/ && cp bin/* /results/
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
echo "Finished at: " `date`