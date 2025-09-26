FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o yahba main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/yahba /usr/local/bin/yahba

ENTRYPOINT ["yahba"]
