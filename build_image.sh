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
CMD /usr/bin/make deps  && /usr/bin/make build && ls -l bin/ && cp bin/* /results/

EOF
docker build -t ${IMG} -f Dockerfile.builder .

# /builds/piersharding/nfsshare-controller
echo "Build ${BIN} in docker"
docker run --name gobuilder \
   -v /builds/piersharding/nfsshare-controller/bin:/results \
   -it ${IMG}
echo "Finished build."
echo "Does the container still exist:"
docker ps -a
#echo "Copy out ${BIN}"
#docker cp gobuilder:/results/${BIN} bin/
echo "Output:"
ls -latr bin/
