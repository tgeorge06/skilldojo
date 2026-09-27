# Single-binary image: Go app + Litestream for continuous SQLite backups.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/skilldojo .

FROM litestream/litestream:0.3.13 AS litestream

FROM alpine:3.22
RUN apk add --no-cache ca-certificates su-exec \
    && addgroup -S app && adduser -S -G app -h /app app
COPY --from=litestream /usr/local/bin/litestream /usr/local/bin/litestream
COPY --from=build /out/skilldojo /app/skilldojo
COPY deploy/litestream.yml /etc/litestream.yml
COPY deploy/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh
EXPOSE 8080
# The entrypoint starts as root only to take ownership of the volume, then
# drops to the unprivileged app user for Litestream and the server.
ENTRYPOINT ["/app/entrypoint.sh"]
