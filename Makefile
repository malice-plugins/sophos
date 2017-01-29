REPO=malice
NAME=sophos
VERSION=$(shell cat VERSION)

build:
	docker build -t $(REPO)/$(NAME):$(VERSION) .

size: build
	sed -i.bu 's/docker image-.*-blue/docker image-$(shell docker images --format "{{.Size}}" $(REPO)/$(NAME):$(VERSION))-blue/' README.md

tags:
	docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" $(REPO)/$(NAME)

test:
	docker run --init --rm $(REPO)/$(NAME):$(VERSION) -t EICAR

.PHONY: build size tags test
