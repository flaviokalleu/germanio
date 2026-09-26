# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o flang .

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /build/flang /usr/local/bin/flang
COPY --from=builder /build/exemplos ./exemplos

EXPOSE 8080

CMD ["flang", "run", "exemplos/loja/inicio.ge"]
