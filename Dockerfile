# Runs a Germanio application. Build the image once, then mount your project:
#
#   docker build -t germanio .
#   docker run --rm -p 8080:8080 -v "$PWD/my_app:/app" germanio
#
# The application's data (SQLite) lives in /data; mount a volume there to keep it.
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/ge ./cmd/ge

FROM alpine:3.20
RUN apk add --no-cache ca-certificates git && adduser -D -h /app germanio && mkdir /data && chown germanio /data
COPY --from=builder /out/ge /usr/local/bin/ge
USER germanio
WORKDIR /app
ENV GERMANIO_SQLITE=/data/app.db GERMANIO_PRODUCAO=1
VOLUME ["/data"]
EXPOSE 8080
CMD ["ge", "run", "app.ge", "8080"]
