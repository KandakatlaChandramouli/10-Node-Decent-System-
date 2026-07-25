FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o sovereign-node ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/sovereign-node .

EXPOSE 9090 9095

ENTRYPOINT ["./sovereign-node"]
