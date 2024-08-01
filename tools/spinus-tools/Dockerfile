# syntax=docker/dockerfile:1

FROM golang:1.22-bookworm AS builder
WORKDIR /app

RUN go env -w GOBIN=/app
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
RUN go install github.com/a-h/templ/cmd/templ@latest
COPY . ./
RUN go build -o spinus-sqlc-generate ./tools/spinus-tools

FROM debian:stable-slim
WORKDIR /app
COPY --from=builder /app /app
CMD ["sleep", "infinity"]
