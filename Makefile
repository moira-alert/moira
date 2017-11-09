GIT_HASH := $(shell git log --pretty=format:%H -n 1)
GIT_TAG := $(shell git describe --always --tags --abbrev=0 | tail -c+2)
GIT_COMMIT := $(shell git rev-list v${GIT_TAG}..HEAD --count)
GO_VERSION := $(shell go version | cut -d' ' -f3)
VERSION := ${GIT_TAG}.${GIT_COMMIT}
VENDOR := "SKB Kontur"
URL := "https://github.com/moira-alert/moira"
LICENSE := "MIT"

.PHONY: default
default: test build

.PHONY: prepare
prepare:
	go get github.com/kardianos/govendor
	govendor sync

.PHONY: lint
lint: prepare
	go get github.com/alecthomas/gometalinter
	gometalinter --install
	gometalinter ./... --vendor --skip mock --disable=errcheck --disable=gocyclo --deadline=5m

.PHONY: test
test: prepare
	echo 'mode: atomic' > coverage.txt && go list ./... | grep -v "/vendor/" | xargs -n1 -I{} sh -c 'go test -v -bench=. -covermode=atomic -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> coverage.txt' && rm coverage.tmp

.PHONY: build
build: prepare
	for service in "filter" "notifier" "api" "checker" "cli" ; do \
		CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.Version=${VERSION} -X main.GoVersion=${GO_VERSION} -X main.GitHash=${GIT_HASH}" -o build/$$service github.com/moira-alert/moira/cmd/$$service ; \
	done

.PHONY: clean
clean:
	rm -rf build

.PHONY: tar
tar:
	for service in "filter" "notifier" "api" "checker" "cli" ; do \
		mkdir -p build/root/$$service/usr/bin ; \
		mkdir -p build/root/$$service/usr/lib/systemd/system ; \
		mkdir -p build/root/$$service/etc/moira ; \
		cp build/$$service build/root/$$service/usr/bin/moira-$$service ; \
		cp pkg/$$service/moira-$$service.service build/root/$$service/usr/lib/systemd/system/moira-$$service.service ; \
		cp pkg/$$service/$$service.yml build/root/$$service/etc/moira/$$service.yml ; \
	done
	cp pkg/filter/storage-schemas.conf build/root/filter/etc/moira/storage-schemas.conf
	for service in "filter" "notifier" "api" "checker" "cli" ; do \
		tar -czvPf build/moira-$$service-${VERSION}.tar.gz -C build/root/$$service . ; \
	done

.PHONY: rpm
rpm: tar
	for service in "notifier" "api" "checker" "cli" ; do \
		fpm -t rpm \
			-s "tar" \
			--description "Moira $$service" \
			--vendor ${VENDOR} \
			--url ${URL} \
			--license ${LICENSE} \
			--name "moira-$$service" \
			--version "${VERSION}" \
			--iteration "1" \
			--config-files "/etc/moira/$$service.yml" \
			--after-install "./pkg/$$service/postinst" \
			-p build \
			build/moira-$$service-${VERSION}.tar.gz ; \
	done
	fpm -t rpm \
		-s "tar" \
		--description "Moira filter" \
		--vendor ${VENDOR} \
		--url ${URL} \
		--license ${LICENSE} \
		--name "moira-filter" \
		--version "${VERSION}" \
		--iteration "1" \
		--config-files "/etc/moira/filter.yml" \
		--config-files "/etc/moira/storage-schemas.conf" \
		--after-install "./pkg/filter/postinst" \
		-p build \
		build/moira-filter-${VERSION}.tar.gz

.PHONY: deb
deb: tar
	for service in "notifier" "api" "checker" "cli" ; do \
		fpm -t deb \
			-s "tar" \
			--description "Moira $$service" \
			--vendor ${VENDOR} \
			--url ${URL} \
			--license ${LICENSE} \
			--name "moira-$$service" \
			--version "${VERSION}" \
			--iteration "1" \
			--config-files "/etc/moira/$$service.yml" \
			--after-install "./pkg/$$service/postinst" \
			-p build \
			build/moira-$$service-${VERSION}.tar.gz ; \
	done
	fpm -t deb \
		-s "tar" \
		--description "Moira filter" \
		--vendor ${VENDOR} \
		--url ${URL} \
		--license ${LICENSE} \
		--name "moira-filter" \
		--version "${VERSION}" \
		--iteration "1" \
		--config-files "/etc/moira/filter.yml" \
		--config-files "/etc/moira/storage-schemas.conf" \
		--after-install "./pkg/filter/postinst" \
		-p build \
		build/moira-filter-${VERSION}.tar.gz

.PHONY: packages
packages: clean build tar rpm deb

.PHONY: docker_image
docker_image:
	for service in "filter" "notifier" "api" "checker" ; do \
		docker build -f Dockerfile.$$service -t moira/$$service:${VERSION} -t moira/$$service:latest . ; \
	done

.PHONY: docker_push
docker_push:
	for service in "filter" "notifier" "api" "checker" ; do \
		docker push moira/$$service:latest ; \
	done

.PHONY: docker_push_release
docker_push_release:
	for service in "filter" "notifier" "api" "checker" ; do \
		docker push moira/$$service:${VERSION} ; \
	done
