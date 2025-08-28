FROM golang:1.24-alpine AS builder

WORKDIR /app

# Настройка прокси и IPv4 для сборки
ENV GOPROXY=https://proxy.golang.org,direct
ENV GODEBUG=netdns=go

# Копируем модули и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Собираем бинарник из main.go
RUN go build -o walletgenerator ./main.go

# Финальный минимальный образ
FROM alpine:3.20
WORKDIR /root/

# Копируем бинарник из билдера
COPY --from=builder /app/walletgenerator .

# Монтируем volume для базы
VOLUME ["/addresses"]

CMD ["./walletgenerator"]
