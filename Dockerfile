# --- Build Stage ---
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

# Automatically injected by Buildx
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go install github.com/pressly/goose/v3/cmd/goose@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /app/bin/sound-stage-backend ./cmd/main.go

# --- Runtime Stage ---
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY --from=builder /go/bin/linux_${TARGETARCH}/goose /usr/local/bin/goose
COPY --from=builder /app/bin/sound-stage-backend .
COPY --from=builder /app/internal/infra/database/migrations ./migrations

USER appuser

EXPOSE 8000

CMD ["./sound-stage-backend"]
