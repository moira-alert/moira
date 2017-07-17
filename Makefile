VERSION := $(shell git describe --always --tags --abbrev=0 | tail -c +2)
RELEASE := $(shell git describe --always --tags | awk -F- '{ if ($$2) dot="."} END { printf "1%s%s%s%s\n",dot,$$2,dot,$$3}')
VENDOR := "SKB Kontur"
URL := "https://github.com/moira-alert"
LICENSE := "GPLv3"

.PHONY: test prepare build tar rpm deb

default: test build

prepare:
	go get github.com/kardianos/govendor
	govendor sync
	go get github.com/alecthomas/gometalinter
	gometalinter --install

lint: prepare
	gometalinter ./... --vendor --skip mock --disable=errcheck --disable=gocyclo

test: prepare
	echo 'mode: atomic' > coverage.txt && go list ./... | grep -v "/vendor/" | xargs -n1 -I{} sh -c 'go test -bench=. -covermode=atomic -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> coverage.txt' && rm coverage.tmp
W
build:
	go build -ldflags "-X main.Version=$(VERSION)-$(RELEASE)" -o build/moira-notifier github.com/moira-alert/moira-alert/cmd/notifier
	go build -ldflags "-X main.Version=$(VERSION)-$(RELEASE)" -o build/moira-cache github.com/moira-alert/moira-alert/cmd/cache

clean:
	rm -rf build

tar:
	mkdir -p build/root/usr/bin
	mkdir -p build/root/usr/lib/systemd/system
	mkdir -p build/root/etc/logrotate.d/
	mkdir -p build/root/etc/moira
	mkdir -p build/root/usr/lib/tmpfiles.d

	mv build/moira-notifier build/root/usr/bin/
	cp pkg/notifier/moira-notifier.service build/root/usr/lib/systemd/system/moira-notifier.service
	cp pkg/notifier/logrotate build/root/etc/logrotate.d/moira-notifier

	mv build/moira-cache build/root/usr/bin/
	cp pkg/cache/moira-cache.service build/root/usr/lib/systemd/system/moira-cache.service
	cp pkg/cache/logrotate build/root/etc/logrotate.d/moira-cache
	cp pkg/cache/storage-schemas.conf build/root/etc/moira/storage-schemas.conf
	cp pkg/cache/cache.yml build/root/etc/moira/cache.yml
	cp pkg/cache/tmpfiles build/root/usr/lib/tmpfiles.d/moira.conf

	tar -czvPf build/moira-$(VERSION)-$(RELEASE).tar.gz -C build/root .

rpm: tar
	fpm -t rpm \
		-s "tar" \
		--description "Moira" \
		--vendor $(VENDOR) \
		--url $(URL) \
		--license $(LICENSE) \
		--name "moira" \
		--version "$(VERSION)" \
		--iteration "$(RELEASE)" \
		--after-install "./pkg/postinst" \
		--depends logrotate \
		-p build \
		build/moira-$(VERSION)-$(RELEASE).tar.gz

deb: tar
	fpm -t deb \
		-s "tar" \
		--description "Moira" \
		--vendor $(VENDOR) \
		--url $(URL) \
		--license $(LICENSE) \
		--name "moira" \
		--version "$(VERSION)" \
		--iteration "$(RELEASE)" \
		--after-install "./pkg/postinst" \
		--depends logrotate \
		-p build \
		build/moira-$(VERSION)-$(RELEASE).tar.gz

packages: clean build tar rpm deb
