FROM --platform=$BUILDPLATFORM docker.io/tonistiigi/xx:1.9.0 AS xx

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27.0-trixie AS builder
COPY --from=xx / /
ARG TARGETPLATFORM
WORKDIR /go/src/app
ADD . /go/src/app

RUN apt-get update && apt-get install -y clang lld && xx-apt-get install -y xx-c-essentials
RUN xx-go --wrap
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=secret,id=certificates,target=/etc/ssl/certs/ca-certificates.crt \
    make bin/controller
RUN xx-verify bin/controller && go version -m bin/controller

FROM docker.io/chainguard/glibc-dynamic:latest
COPY --from=builder /go/src/app/bin/controller /bin
ENTRYPOINT ["/bin/controller"]
