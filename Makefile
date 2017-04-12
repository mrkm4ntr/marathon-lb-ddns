NAME := marathon-lb-ddns

DOCKER_IMAGE_NAME := mrkm4ntr/marathon-lb-ddns
TAG  ?= latest
DOCKER_IMAGE      := $(DOCKER_IMAGE_NAME):$(TAG)

.PHONY: clean
clean:
	rm -rf bin/*
	rm -rf dist/*
	rm -rf vendor/*

.PHONY: deps
deps:
	glide install

.PHONY: test
test: build
	go test

.PHONY: build
build: deps
	env GOOS=linux GOARCH=386 go build -o dist/$(NAME)

.PHONY: docker-build
docker-build: build
	docker build -t $(DOCKER_IMAGE) .

.PHONY: publish
publish: docker-build
	docker push $(DOCKER_IMAGE)
