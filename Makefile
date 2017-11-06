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
	gometalinter ./... --vendor --skip mock --disable=errcheck --disable=gocyclo --deadline=5m

test: prepare
	echo 'mode: atomic' > coverage.txt && go list ./... | grep -v "/vendor/" | xargs -n1 -I{} sh -c 'go test -v -bench=. -covermode=atomic -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> coverage.txt' && rm coverage.tmp

build:
	for service in "filter" "notifier" "api" "checker" "cli" ; do \
		CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.Version=${VERSION}-${RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitHash=${GIT_HASH}" -o build/$$service github.com/moira-alert/moira/cmd/$$service ; \
	done

clean:
	rm -rf build

tar:
	for service in "filter" "notifier" "api" "checker" "cli" ; do \
		mkdir -p build/root/$$service/usr/bin ; \
		mkdir -p build/root/$$service/usr/lib/systemd/system ; \
		mkdir -p build/root/$$service/etc/moira ; \
		cp build/$$service build/root/$$service/usr/bin/moira-$$service ; \
		cp pkg/$$service/moira-$$service.service build/root/$$service/usr/lib/systemd/system/moira-$$service.service ; \
		cp pkg/storage-schemas.conf build/root/$$service/etc/moira/storage-schemas.conf ; \
		cp pkg/$$service/$$service.yml build/root/$$service/etc/moira/$$service.yml ; \
		tar -czvPf build/moira-$$service-${VERSION}-${RELEASE}.tar.gz -C build/root/$$service . ; \
	done

rpm: tar
	for service in "filter" "notifier" "api" "checker" "cli" ; do \
		fpm -t rpm \
			-s "tar" \
			--description "Moira $$service" \
			--vendor ${VENDOR} \
			--url ${URL} \
			--license ${LICENSE} \
			--name "moira-$$service" \
			--version "${VERSION}" \
			--iteration "${RELEASE}" \
			--config-files "/etc/moira/$$service.yml" \
			--config-files "/etc/moira/storage-schemas.conf" \
			--after-install "./pkg/$$service/postinst" \
			-p build \
			build/moira-$$service-${VERSION}-${RELEASE}.tar.gz ; \
	done

deb: tar
	for service in "filter" "notifier" "api" "checker" "cli" ; do \
		fpm -t deb \
			-s "tar" \
			--description "Moira $$service" \
			--vendor ${VENDOR} \
			--url ${URL} \
			--license ${LICENSE} \
			--name "moira-$$service" \
			--version "${VERSION}" \
			--iteration "${RELEASE}" \
			--config-files "/etc/moira/$$service.yml" \
			--config-files "/etc/moira/storage-schemas.conf" \
			--after-install "./pkg/$$service/postinst" \
			-p build \
			build/moira-$$service-${VERSION}-${RELEASE}.tar.gz ; \
	done

packages: clean build tar rpm deb

docker_image:
	docker build -t ${IMAGE_NAME}:${VERSION} -t ${IMAGE_NAME}:latest .

docker_push:
	docker push ${IMAGE_NAME}:latest

docker_push_release:
	docker push ${IMAGE_NAME}:latest
	docker push ${IMAGE_NAME}:${VERSION}
