FROM golang as builder
WORKDIR /app
RUN go env -w GOBIN=/app
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
COPY . ./
RUN go build -o spinus-sqlc-generate ./tools/sqlc


CMD ["sleep", "infinity"]
