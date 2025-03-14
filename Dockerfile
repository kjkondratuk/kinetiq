FROM golang:1.24.1 AS builder

WORKDIR /app

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -o kinetiq main.go

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/kinetiq .

EXPOSE 8080

CMD ["./kinetiq"]