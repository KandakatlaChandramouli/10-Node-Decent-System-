FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o /app/sovereignd cmd/sovereignd/main.go

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/sovereignd /app/sovereignd
COPY config/config.yaml /app/config/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/sovereignd"]
