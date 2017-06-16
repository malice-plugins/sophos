REPO=malice
NAME=sophos
VERSION=$(shell cat VERSION)

all: gotest build size test

build:
	docker build -t $(REPO)/$(NAME):$(VERSION) .

dev:
	docker build -f Dockerfile.dev -t $(REPO)/$(NAME):$(VERSION) .

size:
	sed -i.bu 's/docker%20image-.*-blue/docker%20image-$(shell docker images --format "{{.Size}}" $(REPO)/$(NAME):$(VERSION)| cut -d' ' -f1)%20GB-blue/' README.md

tags:
	docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" $(REPO)/$(NAME)

tar:
	docker save $(REPO)/$(NAME):$(VERSION) -o $(NAME).tar

gotest:
	go get
	go test -v

avtest:
	@echo "===> Sophos Version"
	@docker run --init --rm --entrypoint=bash $(REPO)/$(NAME):$(VERSION) -c "/opt/sophos/bin/savscan --version"
	@echo "===> Sophos EICAR Test"
	@docker run --init --rm --entrypoint=bash $(REPO)/$(NAME):$(VERSION) -c "/opt/sophos/bin/savscan -f -ss EICAR" || true

test:
	docker run --init --rm $(REPO)/$(NAME):$(VERSION)
	docker run --init --rm $(REPO)/$(NAME):$(VERSION) -V EICAR > results.json
	cat results.json | jq .

.PHONY: build dev size tags test gotest
