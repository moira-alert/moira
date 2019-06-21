FROM golang:1.12.6 as builder

RUN go get github.com/kardianos/govendor

COPY ./vendor/vendor.json /go/src/github.com/moira-alert/moira/vendor/vendor.json
WORKDIR /go/src/github.com/moira-alert/moira
RUN govendor sync

COPY . /go/src/github.com/moira-alert/moira/

CMD CGO_ENABLED=0 go test ./...
