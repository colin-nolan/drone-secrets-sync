FROM golang:alpine as builder

RUN apk add --no-cache make

WORKDIR /build

# Separate for caching
COPY go.mod go.sum ./
RUN go mod download

COPY Makefile .
COPY pkg pkg
COPY cmd cmd
RUN make build


FROM alpine

COPY --from=builder /build/bin/drone-secrets-manager /usr/local/bin/drone-secrets-manager

ENTRYPOINT ["/usr/local/bin/drone-secrets-manager"]
