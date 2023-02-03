# syntax = docker/dockerfile:1.4
FROM golang:1.19-alpine as builder
COPY . /go/src/github.com/joyme123/gnt
WORKDIR /go/src/github.com/joyme123/gnt
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod \
  go build -o gnt main.go

FROM alpine:3.17.1
COPY --from=builder /go/src/github.com/joyme123/gnt/gnt /usr/bin