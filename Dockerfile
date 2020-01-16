FROM golang:1.13.5 AS BuildStage

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on
ENV GOPROXY=off
ENV GOFLAGS=-mod=vendor

COPY . /go/src/mine_and_yours
WORKDIR /go/src/mine_and_yours/

RUN go build -installsuffix cgo -o /mine_and_yours /go/src/mine_and_yours/ && \
    go clean -i -cache -testcache


FROM ubuntu:bionic-20191202

WORKDIR /

RUN export DEBIAN_FRONTEND=noninteractive && \
    apt update && apt upgrade -y && \
    apt install ca-certificates curl dnsutils tzdata \
    && update-ca-certificates 2>/dev/null || true
COPY --from=BuildStage /mine_and_yours /

CMD ["/mine_and_yours", "-listen", "[::]:80"]
