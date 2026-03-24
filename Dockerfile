# Bould Satge
FROM docker.io/library/golang:1.25.6-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o main ./cmd/main.go

# Run Stage
FROM docker.io/library/alpine:3.23.3
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
