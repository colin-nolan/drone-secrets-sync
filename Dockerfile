FROM docker.io/golang:alpine as builder

ARG TARGETARCH
RUN echo $TARGETARCH

RUN apk add --no-cache bash make

WORKDIR /build

# Separate for caching
COPY go.mod go.sum ./
RUN go mod download

COPY Makefile .
COPY pkg pkg
COPY cmd cmd

ARG VERSION=unset
RUN make build VERSION="${VERSION}"

RUN cp "build/release/${VERSION}"/drone-secrets-sync* /drone-secrets-sync


FROM docker.io/alpine

COPY --from=builder /drone-secrets-sync /usr/local/bin/drone-secrets-sync

ENTRYPOINT ["/usr/local/bin/drone-secrets-sync"]
