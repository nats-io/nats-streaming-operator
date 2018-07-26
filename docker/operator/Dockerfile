FROM golang:1.9.0-alpine3.6 AS builder
WORKDIR $GOPATH/src/github.com/nats-io/nats-streaming-operator/
COPY . .
RUN apk add --update git
RUN CGO_ENABLED=0 go build -installsuffix cgo -o /nats-streaming-operator ./cmd/nats-streaming-operator/main.go

FROM alpine:3.6
COPY --from=builder /nats-streaming-operator /usr/local/bin/nats-streaming-operator
CMD ["nats-streaming-operator"]
