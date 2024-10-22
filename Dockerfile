FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOARCH=amd64 GOOS=linux go build -o main .

FROM alpine:latest 

COPY --from=builder /app/main .
COPY .env .

EXPOSE 8080

CMD ["./main"]
