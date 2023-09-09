FROM golang:alpine as builder

RUN apk add --no-cache make

WORKDIR /build

# Separate for caching
COPY go.mod go.sum ./
RUN go mod download

COPY Makefile .
COPY pkg pkg
COPY cmd cmd

ARG VERSION=unset
RUN make build VERSION="${VERSION}"


FROM alpine

COPY --from=builder /build/bin/drone-secrets-sync /usr/local/bin/drone-secrets-sync

ENTRYPOINT ["/usr/local/bin/drone-secrets-sync"]
