FROM alpine:latest

RUN mkdir -p build/checker

COPY checker.yml build/checker

WORKDIR /build/checker

ENTRYPOINT ["/build/checker/checker"]
