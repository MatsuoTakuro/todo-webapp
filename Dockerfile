FROM golang:1.18.2-alpine

# Install live-reload tool
RUN go install github.com/cosmtrek/air@latest

WORKDIR /app

COPY . .

RUN go build
