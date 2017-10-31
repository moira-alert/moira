FROM alpine:latest

COPY build/api /
COPY api.yml /

EXPOSE 8081 8081

WORKDIR /build/api

ENTRYPOINT ["/api"]
