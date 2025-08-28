FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o walletgenerator ./cmd/walletgenerator

FROM alpine:3.20
WORKDIR /root/
COPY --from=builder /app/walletgenerator .

CMD ["./walletgenerator"]
