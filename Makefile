GOSRC = $(shell find . -type f -name '*.go')

VERSION=v1.2.0

build: ddi_controller

ddi_controller: $(GOSRC)
	CGO_ENABLED=0 GOOS=linux go build -o ddi_controller cmd/controller/controller.go

build-image:
	docker build -t linkingthing/ddi-controller:${VERSION} .
	docker image prune -f

docker:
	docker build -t linkingthing/ddi-controller:${VERSION} .
	docker image prune -f
	docker push linkingthing/ddi-controller:${VERSION}

clean:
	rm -rf ddi_controller

clean-image:
	docker rmi linkingthing/ddi-controller:${VERSION}

.PHONY: clean install
