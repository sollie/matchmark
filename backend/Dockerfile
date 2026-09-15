FROM golang:1.26.3-alpine3.23 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /bin/matchmark ./cmd/server

FROM alpine:3.24.1

RUN addgroup -S matchmark && adduser -S -G matchmark matchmark

WORKDIR /app

COPY --from=builder /bin/matchmark /app/matchmark

USER matchmark

EXPOSE 8080

ENTRYPOINT ["/app/matchmark"]
