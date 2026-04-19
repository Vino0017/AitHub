# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o skillhub ./cmd/api

# Run stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/skillhub .
COPY migrations/ migrations/
COPY scripts/ scripts/
COPY skills/ skills/
COPY dist/ dist/

EXPOSE 8080
ENTRYPOINT ["./skillhub"]
