FROM alpine:3.6

RUN adduser -D nats-streaming-operator
USER nats-streaming-operator

ADD tmp/_output/bin/nats-streaming-operator /usr/local/bin/nats-streaming-operator
