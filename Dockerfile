FROM node:24-alpine3.24 AS web
WORKDIR /src/web
COPY source/web/package.json source/web/package-lock.json ./
RUN npm ci
COPY source/web ./
RUN npm run build

FROM golang:1.26-alpine3.24 AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY source/go.mod source/go.sum ./
RUN go mod download
COPY source ./
COPY --from=web /src/web/dist ./web/dist
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/matchmark ./cmd/server
RUN mkdir -p /out/data && chown 65532:65532 /out/data

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/matchmark /matchmark
COPY --from=build --chown=65532:65532 /out/data /data
ENV DB_PATH=/data/matchmark.db
USER 65532:65532
VOLUME /data
EXPOSE 8080 8443
ENTRYPOINT ["/matchmark"]
