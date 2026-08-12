# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src/monitoring
COPY monitoring/go.mod monitoring/go.sum* ./
RUN go mod download
COPY monitoring/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /uploy-monitor .

FROM alpine:3.22
RUN apk add --no-cache docker-cli ca-certificates
COPY --from=build /uploy-monitor /usr/local/bin/uploy-monitor
VOLUME ["/data"]
EXPOSE 9184
HEALTHCHECK --interval=2s --timeout=2s --retries=15 CMD wget -q -O /dev/null http://127.0.0.1:9184/health || exit 1
ENTRYPOINT ["uploy-monitor"]
