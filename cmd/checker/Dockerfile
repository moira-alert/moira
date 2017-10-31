FROM alpine:latest

mkdir -p build/checker

COPY checker.yml build/checker

WORKDIR /build/checker

ENTRYPOINT ["/build/checker/checker"]
