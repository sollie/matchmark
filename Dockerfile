FROM node:24-alpine3.24 AS web
WORKDIR /src/web
COPY source/web/package.json source/web/package-lock.json ./
RUN npm ci
COPY source/web ./
RUN npm run build

FROM golang:1.27.2-alpine3.24 AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY source/go.mod source/go.sum ./
RUN go mod download
COPY source ./
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -tags musl -trimpath -ldflags="-s -w" -o /out/matchmark ./cmd/server

FROM alpine:3.24
RUN apk add --no-cache ca-certificates libgcc \
    && addgroup -g 65532 -S matchmark \
    && adduser -u 65532 -S -G matchmark -H -h /nonexistent matchmark \
    && mkdir -p /data /var/cache/turso \
    && chown matchmark:matchmark /data /var/cache/turso
COPY --from=build /out/matchmark /usr/local/bin/matchmark
COPY LICENSE /usr/share/licenses/matchmark/LICENSE
ENV DB_PATH=/data/matchmark.db \
    TURSO_GO_CACHE_DIR=/var/cache/turso
USER matchmark
VOLUME /data
EXPOSE 8080 8443
ENTRYPOINT ["matchmark"]
