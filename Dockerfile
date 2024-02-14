# syntax=docker/dockerfile:1

FROM golang:1.22-bookworm as builder
ARG CGO_ENABLED=0
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . ./

RUN go build -o spinus ./cmd/server

FROM scratch
WORKDIR /app
COPY --from=builder /app/spinus /app/spinus

EXPOSE 8213

ENTRYPOINT ["/app/spinus"]
