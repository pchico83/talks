# syntax = docker/dockerfile:experimental

FROM golang:1.11-alpine as builder

RUN apk --update add --no-cache --virtual .build-deps \
    alpine-sdk cmake sudo libssh2 \
    libssh2-dev git && \
    rm -rf /var/lib/apt/lists/*

ENV SRC_DIR=/go/src/bitbucket.org/okteto/okteto/backend
ENV CGO_ENABLED=1
ENV GOOS=linux

WORKDIR $SRC_DIR

ADD vendor ./vendor
RUN cd vendor && go install $(go list ./...)

ADD . .
# --mount=type=cache,target=/root/.cache/go-build
RUN go build -installsuffix cgo -o okteto .
