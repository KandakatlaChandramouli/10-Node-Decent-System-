# Build Stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
# COPY go.sum ./ # Uncomment if go.sum is present
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sovereign-node ./cmd/sovereign

# Execution Stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/sovereign-node .
EXPOSE 9095
CMD ["./sovereign-node"]
