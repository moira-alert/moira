GIT_HASH := $(shell git log --pretty=format:%H -n 1)
GIT_TAG := $(shell git describe --always --tags --abbrev=0 | tail -c+2)
GIT_COMMIT := $(shell git rev-list v${GIT_TAG}..HEAD --count)
GO_VERSION := $(shell go version | cut -d' ' -f3)
VERSION := ${GIT_TAG}.${GIT_COMMIT}
IMAGE_NAME := kontur/moira
RELEASE := 1
VENDOR := "SKB Kontur"
URL := "https://github.com/moira-alert"
LICENSE := "GPLv3"

.PHONY: test prepare build tar rpm deb docker_image docker_push docker_push_release

default: test build

prepare:
	go get github.com/kardianos/govendor
	govendor sync

lint: prepare
	go get github.com/alecthomas/gometalinter
	gometalinter --install
	gometalinter ./... --vendor --skip mock --disable=errcheck --disable=gocyclo --deadline=1m

test: prepare
	echo 'mode: atomic' > coverage.txt && go list ./... | grep -v "/vendor/" | xargs -n1 -I{} sh -c 'go test -v -bench=. -covermode=atomic -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> coverage.txt' && rm coverage.tmp

build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.Version=${VERSION}-${RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitHash=${GIT_HASH}" -o build/moira github.com/moira-alert/moira/cmd/moira

clean:
	rm -rf build

tar:
	mkdir -p build/root/usr/bin
	mkdir -p build/root/usr/lib/systemd/system
	mkdir -p build/root/etc/logrotate.d
	mkdir -p build/root/etc/moira

	cp build/moira build/root/usr/bin/
	cp pkg/moira.service build/root/usr/lib/systemd/system/moira.service
	cp pkg/logrotate build/root/etc/logrotate.d/moira

	cp pkg/storage-schemas.conf build/root/etc/moira/storage-schemas.conf
	cp pkg/moira.yml build/root/etc/moira/moira.yml

	tar -czvPf build/moira-${VERSION}-${RELEASE}.tar.gz -C build/root .

rpm: tar
	fpm -t rpm \
		-s "tar" \
		--description "Moira" \
		--vendor ${VENDOR} \
		--url ${URL} \
		--license ${LICENSE} \
		--name "moira" \
		--version "${VERSION}" \
		--iteration "${RELEASE}" \
		--config-files "/etc/moira/moira.yml" \
		--config-files "/etc/moira/storage-schemas.conf" \
		--after-install "./pkg/postinst" \
		--depends logrotate \
		-p build \
		build/moira-${VERSION}-${RELEASE}.tar.gz

deb: tar
	fpm -t deb \
		-s "tar" \
		--description "Moira" \
		--vendor ${VENDOR} \
		--url ${URL} \
		--license ${LICENSE} \
		--name "moira" \
		--version "${VERSION}" \
		--iteration "${RELEASE}" \
		--config-files "/etc/moira/moira.yml" \
		--config-files "/etc/moira/storage-schemas.conf" \
		--after-install "./pkg/postinst" \
		--depends logrotate \
		-p build \
		build/moira-${VERSION}-${RELEASE}.tar.gz

packages: clean build tar rpm deb

docker_image:
	docker build -t ${IMAGE_NAME}:${VERSION} -t ${IMAGE_NAME}:latest .

docker_push:
	docker push ${IMAGE_NAME}:latest

docker_push_release:
	docker push ${IMAGE_NAME}:latest
	docker push ${IMAGE_NAME}:${VERSION}
