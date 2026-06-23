FROM golang:1.26-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 go build -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o /out/weather-cli ./cmd/weather-cli
RUN CGO_ENABLED=0 go build -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o /out/weather-web ./cmd/weather-web
RUN cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /src/web/static/wasm_exec.js
RUN GOOS=js GOARCH=wasm go build -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o /src/web/static/weather-web.wasm ./cmd/weather-web-wasm

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S weather && adduser -S -G weather -h /app weather
WORKDIR /app

COPY --from=builder /out/weather-cli /usr/local/bin/weather-cli
COPY --from=builder /out/weather-web /usr/local/bin/weather-web
COPY --from=builder /src/web/static ./web/static

ENV WEATHER_CLI_WEB_ADDR=:8080 \
    WEATHER_CLI_BIN=/usr/local/bin/weather-cli \
    WEATHER_CLI_STATIC_ROOT=/app/web/static \
    WEATHER_CLI_WEB_CONFIG_DIR=/tmp/weather-cli \
    TZ=Europe/Vienna

USER weather
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/health || exit 1

CMD ["/usr/local/bin/weather-web"]
