.PHONY: build-linux container install-linux-deps
build-linux: install-linux-deps
	CGO_LDFLAGS='./build/dep/linux/libpcap.a' GOOS=linux CGO_ENABLED=1 go build -ldflags '-linkmode "external" -extldflags "-static"' -a -o gnt main.go

install-linux-deps:
	sudo apt install -y make gcc g++ flex bison libpcap-dev

container:
	docker build -t joyme/gnt .
