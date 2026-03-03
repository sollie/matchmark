FROM node:lts-alpine3.23 AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM alpine:3.23

COPY --from=builder /app/build /srv/frontend
