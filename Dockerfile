# Stage 1: Build stage
FROM golang:1.25.5-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

# Stage 2: Run stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/main .
COPY .env.example .env
COPY --from=builder /app/internal/db/migration ./internal/db/migration

EXPOSE 8080

CMD ["./main"]