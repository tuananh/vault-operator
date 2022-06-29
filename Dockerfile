FROM golang:1.18 AS build
WORKDIR /src
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY / ./
RUN --mount=type=cache,target=/go/pkg \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build

FROM alpine:3
RUN apk add --no-cache ca-certificates 
RUN adduser -D nonroot
USER nonroot
COPY --from=build /src/vault-operator /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/vault-operator"]