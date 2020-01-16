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


FROM alpine:3.11.2

WORKDIR /

RUN apk update && apk add --no-cache ca-certificates tzdata && update-ca-certificates 2>/dev/null || true
COPY --from=BuildStage /mine_and_yours /

CMD ["/mine_and_yours", "-listen", "[::]:80"]
