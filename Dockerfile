FROM golang:1.24.2-bookworm as builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o /bin/traffic cmd/main.go

RUN chmod +x /bin/traffic
