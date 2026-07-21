FROM golang:1.22-alpine
WORKDIR /app
COPY main.go .
RUN go build -o sovereign main.go
ENTRYPOINT ["./sovereign"]
