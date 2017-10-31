FROM alpine:latest

RUN mkdir -p build/filter

COPY filter.yml build/filter

EXPOSE 2003 2003

WORKDIR /build/filter

ENTRYPOINT ["/build/filter/filter"]
