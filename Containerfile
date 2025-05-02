FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod tidy

COPY . .

RUN go build -o app ./cmd/main.go

FROM alpine:3.21

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/app .

ENTRYPOINT ["./app"]