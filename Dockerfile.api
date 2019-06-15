FROM golang:1.12.6 as builder

RUN go get github.com/kardianos/govendor

COPY ./vendor/vendor.json /go/src/github.com/moira-alert/moira/vendor/vendor.json
WORKDIR /go/src/github.com/moira-alert/moira
RUN govendor sync

COPY . /go/src/github.com/moira-alert/moira/

ARG GO_VERSION="GoVersion"
ARG GIT_COMMIT="git_Commit"
ARG MoiraVersion="MoiraVersion"

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.MoiraVersion=${MoiraVersion} -X main.GoVersion=${GO_VERSION} -X main.GitCommit=${GIT_COMMIT}" -o build/api github.com/moira-alert/moira/cmd/api


FROM alpine

RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY pkg/api/api.yml /etc/moira/api.yml
COPY pkg/api/web.json /etc/moira/web.json

COPY --from=builder /go/src/github.com/moira-alert/moira/build/api /usr/bin/api
COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/

EXPOSE 8081 8081

ENTRYPOINT ["/usr/bin/api"]
