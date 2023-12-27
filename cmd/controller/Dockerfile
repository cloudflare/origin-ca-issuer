FROM docker.io/library/golang:1.21.5-bookworm AS builder
WORKDIR /go/src/app
ADD . /go/src/app

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=secret,id=certificates,target=/etc/ssl/certs/ca-certificates.crt \
    make bin/controller


FROM gcr.io/distroless/base-nossl-debian12:nonroot
COPY --from=builder /go/src/app/bin/controller /bin
ENTRYPOINT ["/bin/controller"]
