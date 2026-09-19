# syntax=docker/dockerfile:1

# ---------- build ----------
FROM golang:1.25-alpine AS build
WORKDIR /src

# Dependencies go in their own layer, so editing code does not re-download them.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
# CGO_ENABLED=0 produces a static binary, so the runtime image needs no libc.
# -trimpath keeps build paths out of the binary; -s -w drops the debug tables.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---------- run ----------
FROM alpine:3.22
# The hospital HIS is reached over HTTPS, which needs the CA bundle.
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 app
COPY --from=build /out/api /usr/local/bin/api
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
