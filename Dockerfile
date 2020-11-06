FROM golang:1.14.5-alpine3.12 AS build

ENV GOPROXY=https://goproxy.io

RUN mkdir -p /go/src/github.com/linkingthing/ddi-controller
COPY . /go/src/github.com/linkingthing/ddi-controller

WORKDIR /go/src/github.com/linkingthing/ddi-controller
RUN CGO_ENABLED=0 GOOS=linux go build -o ddi-controller cmd/controller/controller.go

FROM alpine:3.12
COPY --from=build /go/src/github.com/linkingthing/ddi-controller/ddi-controller /
ENTRYPOINT ["/ddi-controller"]
