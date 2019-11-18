REPO ?= $(shell go list -m)
VERSION ?= "$(shell git describe --tags 2>/dev/null | sed 's/^v//')"

HUB_IMAGE=nspccdev/neofs

B=\033[0;1m
G=\033[0;92m
R=\033[0m

# Show current version
version:
	@echo $(VERSION)

build: deps
	@printf "${B}${G}⇒ Build binary into ./bin/neofs-cli${R}: "
	@go build \
		-mod=vendor \
		-o ./bin/neofs-cli \
		-ldflags "-w -s -X $(REPO)/Version=$(VERSION) -X $(REPO)/Build=$(BUILD)" ./ \
	&& echo OK || (echo fail && exit 2)

# Make sure that all files added to commit
deps:
	@printf "${B}${G}⇒ Ensure vendor${R}: "
	@go mod tidy -v && echo OK || (echo fail && exit 2)
	@printf "${B}${G}⇒ Download requirements${R}: "
	@go mod download && echo OK || (echo fail && exit 2)
	@printf "${B}${G}⇒ Store vendor localy${R}: "
	@go mod vendor && echo OK || (echo fail && exit 2)

image: deps
	@echo "${B}${G}⇒ Build CLI docker-image ${R}"
	@docker build \
		--build-arg REPO=$(REPO) \
		--build-arg VERSION=$(VERSION) \
		 -f Dockerfile \
		 -t $(HUB_IMAGE)-cli:$(VERSION) .
