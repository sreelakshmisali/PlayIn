# syntax=docker/dockerfile:1

# --- build -------------------------------------------------------------------
FROM golang:1.25-alpine AS build

WORKDIR /src

# Dependencies are copied first so the module cache layer survives source edits.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG VERSION=dev
# CGO is off so the binary is static and can run on a scratch-like base image.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${VERSION}" \
      -o /out/api ./cmd/api

# --- runtime -----------------------------------------------------------------
FROM alpine:3.21 AS runtime

# wget is used by the compose healthcheck; ca-certificates for outbound TLS.
RUN apk add --no-cache ca-certificates wget \
    && addgroup -g 10001 -S app \
    && adduser -u 10001 -S app -G app

WORKDIR /app
COPY --from=build /out/api /app/api

USER app:app
EXPOSE 8080

ENTRYPOINT ["/app/api"]
