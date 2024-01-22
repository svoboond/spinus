# syntax=docker/dockerfile:1

FROM golang as builder
ARG CGO_ENABLED=0
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . ./

RUN go build -o spinus

FROM scratch
WORKDIR /app
COPY --from=builder /app/spinus /app/spinus

EXPOSE 8213

ENTRYPOINT ["/app/spinus"]
