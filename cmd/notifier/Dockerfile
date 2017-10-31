FROM alpine:latest

RUN mkdir -p build/notifier

COPY notifier.yml build/notifier

WORKDIR /build/notifier

ENTRYPOINT ["/build/notifier/notifier"]
