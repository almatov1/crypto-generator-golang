FROM golang:1.24-alpine AS builder

# Устанавливаем gcc и sqlite-dev для сборки с CGO
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Сборка с CGO_ENABLED=1
RUN CGO_ENABLED=1 GOOS=linux go build -o walletgenerator ./main.go

FROM alpine:3.20
WORKDIR /root/

COPY --from=builder /app/walletgenerator .

VOLUME ["/addresses"]

CMD ["./walletgenerator"]
