FROM golang:1.9.1 AS builder

WORKDIR /go/src/github.com/moira-alert/moira
COPY . /go/src/github.com/moira-alert/moira/
RUN go get github.com/kardianos/govendor
RUN govendor sync
#RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.Version=${VERSION}-${RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitHash=${GIT_HASH}" -o build/moira github.com/moira-alert/moira/cmd/moira
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.Version=${VERSION}-${RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitHash=${GIT_HASH}" -o build/api github.com/moira-alert/moira/cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.Version=${VERSION}-${RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitHash=${GIT_HASH}" -o build/filter github.com/moira-alert/moira/cmd/filter
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.Version=${VERSION}-${RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitHash=${GIT_HASH}" -o build/notifier github.com/moira-alert/moira/cmd/notifier
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.Version=${VERSION}-${RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitHash=${GIT_HASH}" -o build/checker github.com/moira-alert/moira/cmd/checker
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.Version=${VERSION}-${RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitHash=${GIT_HASH}" -o build/moira-cli github.com/moira-alert/moira/cmd/moira-cli


FROM alpine

RUN apk add --no-cache ca-certificates && update-ca-certificates

RUN mkdir /config-api
RUN mkdir /config-checker
RUN mkdir /config-filter
RUN mkdir /config-notifier
RUN mkdir -p /usr/local/go/lib/time/

COPY pkg/moira.yml /
COPY pkg/api.yml /config-api/config.yml
COPY pkg/checker.yml /config-checker/config.yml
COPY pkg/filter.yml /config-filter/config.yml
COPY pkg/notifier.yml /config-notifier/config.yml
COPY pkg/storage-schemas.conf /

WORKDIR /

COPY --from=builder /go/src/github.com/moira-alert/moira/build/moira .
COPY --from=builder /go/src/github.com/moira-alert/moira/build/api .
COPY --from=builder /go/src/github.com/moira-alert/moira/build/checker .
COPY --from=builder /go/src/github.com/moira-alert/moira/build/filter .
COPY --from=builder /go/src/github.com/moira-alert/moira/build/notifier .
COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/

# relay
EXPOSE 2003 2003
# api
EXPOSE 8081 8081

ENTRYPOINT ["/moira"]
