# syntax = docker/dockerfile:1.4
FROM golang:1.19-buster as builder
COPY . /go/src/github.com/joyme123/gnt
WORKDIR /go/src/github.com/joyme123/gnt

RUN apt update && apt install -y make gcc g++ flex bison libpcap-dev && wget http://www.tcpdump.org/release/libpcap-1.9.1.tar.gz && tar xvf libpcap-1.9.1.tar.gz

RUN cd libpcap-1.9.1 && ./configure --enable-dbus=no --enable-shared=no && make

RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod \
  	C_INCLUDE_PATH=$C_INCLUDE_PATH:/go/src/github.com/joyme123/gnt/libpcap-1.9.1/ CGO_LDFLAGS='/go/src/github.com/joyme123/gnt/libpcap-1.9.1/libpcap.a' CGO_ENABLED=1 go build -ldflags '-linkmode "external" -extldflags "-static"' -a -o gnt main.go

FROM debian:buster
COPY --from=builder /go/src/github.com/joyme123/gnt/gnt /usr/bin
