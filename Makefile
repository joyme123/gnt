.PHONY: build container run
build:
	go build -o gnt main.go

container:
	docker build -t joyme/gnt .
run:
	docker run -it --rm joyme/gnt:latest bash
