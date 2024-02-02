FROM golang as builder
WORKDIR /app
RUN go env -w GOBIN=/app
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest


FROM debian:stable-slim
WORKDIR /app
COPY --from=builder /app/sqlc /app/sqlc
CMD ["./sqlc", "generate"]
