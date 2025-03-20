FROM golang:1.24.1 AS builder

WORKDIR /app

COPY . .

RUN go build -o kinetiq main.go

FROM alpine:3

WORKDIR /root/
COPY --from=builder /app/kinetiq .

EXPOSE 8080

CMD ["./kinetiq"]