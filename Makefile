GOSRC = $(shell find . -type f -name '*.go')

VERSION=v1.2.0

build: web_controller

web_controller: $(GOSRC)
	CGO_ENABLED=0 GOOS=linux go build -o web_controller cmd/controller/controller.go

build-image:
	docker build -t linkingthing/web-controller:${VERSION} .
	docker image prune -f

docker:
	docker build -t linkingthing/web-controller:${VERSION} .
	docker image prune -f
	docker push linkingthing/web-controller:${VERSION}

clean:
	rm -rf web_controller

clean-image:
	docker rmi linkingthing/web-controller:${VERSION}

.PHONY: clean install
