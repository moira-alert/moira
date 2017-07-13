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
	go test . -coverprofile cover.coverprofile
	go test ./integration_tests/notifier
	go test ./database/redis -coverprofile cover.coverprofile
	go test ./notifier -coverprofile cover.coverprofile
	go test ./notifier/selfstate -coverprofile cover.coverprofile
	go test ./notifier/events -coverprofile cover.coverprofile
	go test ./notifier/notifications -coverprofile cover.coverprofile

build:
	go build -ldflags "-X main.Version=$(VERSION)-$(RELEASE)" -o build/moira-notifier github.com/moira-alert/moira-alert/cmd/notifier

clean:
	rm -rf build

tar:
	mkdir -p build/root/usr/local/bin
	mkdir -p build/root/usr/lib/systemd/system
	mkdir -p build/root/etc/logrotate.d/

	mv build/moira-notifier build/root/usr/local/bin/
	cp pkg/moira-notifier.service build/root/usr/lib/systemd/system/moira-notifier.service
	cp pkg/logrotate build/root/etc/logrotate.d/moira-notifier

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
