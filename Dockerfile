# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ARG VERSION=docker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o /psn-add-api ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /psn-add-admin ./cmd/admin

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=builder /psn-add-api ./psn-add-api
COPY --from=builder /psn-add-admin ./psn-add-admin

EXPOSE 8890

ENTRYPOINT ["/app/psn-add-api"]
