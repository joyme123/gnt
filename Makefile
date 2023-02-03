.PHONY: build container run
build:
	go build -o gnt main.go

build-windows:
	CGO_ENABLED=0 GOOS=windows go build -o gnt.exe main.go

container:
	docker build -t joyme/gnt .
run:
	docker run -it --rm joyme/gnt:latest sh
