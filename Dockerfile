FROM golang:1.9 AS builder

WORKDIR /go/src/github.com/moira-alert/moira
COPY . /go/src/github.com/moira-alert/moira/
RUN go get github.com/kardianos/govendor
#RUN govendor sync
#RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.Version=${VERSION}-${RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitHash=${GIT_HASH}" -o build/api github.com/moira-alert/moira/cmd/api

FROM alpine

RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY pkg/moira.yml /
COPY pkg/storage-schemas.conf /
COPY build/moira /
COPY build/api /
COPY build/checker /

# relay
EXPOSE 2003 2003
# api
EXPOSE 8081 8081

ENTRYPOINT ["/moira"]
