FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go tool templ generate
RUN go build -o squiggle ./cmd

FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/squiggle .
EXPOSE 8080
CMD ["./squiggle"]
