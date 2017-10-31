FROM alpine:latest

mkdir -p build/api

COPY api.yml build/api

EXPOSE 8081 8081

WORKDIR /build/api

ENTRYPOINT ["/build/api/api"]
