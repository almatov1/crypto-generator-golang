FROM golang:1.24.6-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o crypto-generator main.go

CMD ["./crypto-generator"]
