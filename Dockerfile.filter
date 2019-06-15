FROM golang:1.12.6 as builder

RUN go get github.com/kardianos/govendor

COPY ./vendor/vendor.json /go/src/github.com/moira-alert/moira/vendor/vendor.json
WORKDIR /go/src/github.com/moira-alert/moira
RUN govendor sync

COPY . /go/src/github.com/moira-alert/moira/

ARG GO_VERSION="GoVersion"
ARG GIT_COMMIT="git_Commit"
ARG MoiraVersion="MoiraVersion"

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.MoiraVersion=${MoiraVersion} -X main.GoVersion=${GO_VERSION} -X main.GitCommit=${GIT_COMMIT}" -o build/filter github.com/moira-alert/moira/cmd/filter


FROM alpine

RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY pkg/filter/filter.yml /etc/moira/filter.yml
COPY pkg/filter/storage-schemas.conf /etc/moira/storage-schemas.conf

COPY --from=builder /go/src/github.com/moira-alert/moira/build/filter /usr/bin/filter

EXPOSE 2003 2003

ENTRYPOINT ["/usr/bin/filter"]
