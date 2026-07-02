# --- Build Stage ---
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/sound-stage-backend ./cmd/main.go

# --- Runtime Stage ---
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/bin/sound-stage-backend .
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /app/entrypoint.sh .
COPY --from=builder /app/internal/infra/database/migrations ./migrations

RUN chmod +x entrypoint.sh

USER appuser

EXPOSE 8000

ENTRYPOINT ["./entrypoint.sh"]

CMD ["./sound-stage-backend"]
