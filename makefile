REV := $(shell git rev-parse --short HEAD)
OUT := bin/btcount
ENV ?= development

migrate:
	go run cmd/migrator/main.go
.PHONY: migrate

run: build
	${OUT}
.PHONY: run

build: clean
	go build\
		-ldflags '-w -s -X "main.revision=${REV}" -X "main.environment=${ENV}"'\
		-o ${OUT}\
		cmd/btcount/*.go
.PHONY: build

release: export ENV:=production
release: build
.PHONY: release

clean:
	rm -rf ${OUT}
.PHONY: clean

image:
	docker build -t btcount:latest -f .Dockerfile .
.PHONY: image

test:
	go test -count=1 -race -cover ./internal/...
