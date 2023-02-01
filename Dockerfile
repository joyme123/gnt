# syntax = docker/dockerfile:1.4
FROM golang:1.19 as builder
COPY . /go/src/github.com/joyme123/gnt
WORKDIR /go/src/github.com/joyme123/gnt
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod \
  make build

FROM hub.byted.org/infcplibrary/debian:stretch-curl
COPY --from=builder /go/src/github.com/joyme123/gnt/gnt /usr/bin