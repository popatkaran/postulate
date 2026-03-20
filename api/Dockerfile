# Stage 1 — builder
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Cache dependency layer before copying source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X github.com/postulate/api/internal/handler.version=${VERSION} \
              -X github.com/postulate/api/internal/handler.commit=${COMMIT} \
              -X github.com/postulate/api/internal/handler.buildTime=${BUILD_TIME}" \
    -o /build/postulate-api \
    ./cmd/api/...

# Stage 2 — final
FROM gcr.io/distroless/static-debian12

COPY --from=builder /build/postulate-api /postulate-api

USER nonroot:nonroot

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
    CMD ["/postulate-api", "-healthcheck"]

ENTRYPOINT ["/postulate-api"]
