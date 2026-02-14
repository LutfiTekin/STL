# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./
# RUN go mod download # No external dependencies yet

COPY . .

RUN go build -o tram-predictor ./cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk add --no-cache tzdata

WORKDIR /app

COPY --from=builder /app/tram-predictor .
COPY web ./web

EXPOSE 8080

CMD ["./tram-predictor"]
