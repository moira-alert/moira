MARK_NIGHTLY := "nightly"
MARK_UNSTABLE := "unstable"

GIT_BRANCH := "unknown"
GIT_HASH := $(shell git log --pretty=format:%H -n 1)
GIT_HASH_SHORT := $(shell echo "${GIT_HASH}" | cut -c1-7)
GIT_TAG := $(shell git describe --always --tags --abbrev=0 | tail -c+2)
GIT_COMMIT := $(shell git rev-list v${GIT_TAG}..HEAD --count)
GIT_COMMIT_DATE := $(shell git show -s --format=%ci | cut -d\  -f1)

VERSION_FEATURE := ${GIT_TAG}-$(shell echo $(GIT_BRANCH) | cut -c1-100).${GIT_COMMIT_DATE}.${GIT_HASH_SHORT}
VERSION_NIGHTLY := ${GIT_COMMIT_DATE}.${GIT_HASH_SHORT}
VERSION_RELEASE := ${GIT_TAG}.${GIT_COMMIT}

GO_VERSION := 1.23
GO_PATH := $(shell go env GOPATH)
GO111MODULE := on
GOLANGCI_LINT_VERSION := ""

VENDOR := "SKB Kontur"
URL := "https://github.com/moira-alert/moira"
LICENSE := "MIT"

SERVICES := "notifier" "api" "checker" "cli"

.PHONY: default
default: test build

.PHONY: install-lint
install-lint:
	# The recommended way to install golangci-lint into CI/CD
	wget -O - -q https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GO_PATH}/bin ${GOLANGCI_LINT_VERSION}

.PHONY: lint
lint:
	golangci-lint run

.PHONY: mock
mock:
	. ./generate_mocks.sh

.PHONY: install-swag-v2
install-swag-v2:
	go install github.com/swaggo/swag/cmd/swag@v1.16.3

.PHONY: spec-v2
spec-v2:
	echo "Generating Swagger documentation open-api v2"
	swag init -g api/handler/handler.go --output ./docs/v2
	swag fmt

.PHONY: validate-spec-v2
validate-spec-v2:
	openapi-generator-cli validate -i docs/v2/swagger.yaml

.PHONY: install-swag-v3
install-swag-v3:
	go install github.com/swaggo/swag/v2/cmd/swag@v2.0.0-rc4

OS := $(shell uname)

.PHONY: spec-v3
spec-v3:
	@echo "Generating Swagger documentation in open-api v3"
	swag init -g api/handler/handler.go --v3.1
	@echo "Patching generated swagger.yaml with deepObject/explode for search tags as ?tag[1]=val1&tag[2]=val2 (swag cannot generate it by comments)"
	@awk '\
		{ \
			print $$0; \
			if ($$0 ~ /description: Search tag/) tag_found=1; \
			else if (tag_found && $$0 ~ /in: query/) { \
				print "        style: deepObject"; \
				print "        explode: true"; \
				tag_found=0; \
			} \
		} \
	' docs/swagger.yaml > docs/swagger_patched.yaml && mv docs/swagger_patched.yaml docs/swagger.yaml
	@echo "Swagger UI does not support openapi 3.1.0, downgrade version"
ifeq ($(OS), Darwin)
	sed -i '' 's/"openapi": "3.1.0"/"openapi": "3.0.0"/' docs/docs.go
	sed -i '' 's/"openapi": "3.1.0"/"openapi": "3.0.0"/' docs/swagger.json
	sed -i '' 's/"openapi": "3.1.0"/"openapi": "3.0.0"/' docs/swagger.yaml
else
	sed -i 's/"openapi": "3.1.0"/"openapi": "3.0.0"/' docs/docs.go
	sed -i 's/"openapi": "3.1.0"/"openapi": "3.0.0"/' docs/swagger.json
	sed -i 's/"openapi": "3.1.0"/"openapi": "3.0.0"/' docs/swagger.yaml
endif
	swag fmt

.PHONY: validate-spec-v3
validate-spec-v3:
	openapi-generator-cli validate -i docs/swagger.yaml

.PHONY: test
test:
	echo 'mode: atomic' > coverage.txt && go list ./... | xargs -n1 -I{} sh -c 'go test -race -v -bench=. -covermode=atomic -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> coverage.txt' && rm coverage.tmp

.PHONY: ci-test
ci-test:
	echo 'mode: atomic' > coverage.txt && go list ./... | xargs -n1 -I{} sh -c 'go test -race -v -covermode=atomic -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> coverage.txt' && rm coverage.tmp

.PHONY: build
build:
	for service in "filter" $(SERVICES) ; do \
		CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.MoiraVersion=${VERSION_RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitCommit=${GIT_HASH}" -o build/$$service github.com/moira-alert/moira/cmd/$$service ; \
	done

.PHONY: clean
clean:
	rm -rf build

.PHONY: tar
tar:
	for service in "filter" $(SERVICES) ; do \
		mkdir -p build/root/$$service/usr/bin ; \
		mkdir -p build/root/$$service/etc/moira ; \
		cp build/$$service build/root/$$service/usr/bin/moira-$$service ; \
		cp pkg/$$service/$$service.yml build/root/$$service/etc/moira/$$service.yml ; \
	done
	for service in "filter" "notifier" "api" "checker" ; do \
		mkdir -p build/root/$$service/usr/lib/systemd/system ; \
		cp pkg/$$service/moira-$$service.service build/root/$$service/usr/lib/systemd/system/moira-$$service.service ; \
	done
	cp pkg/filter/storage-schemas.conf build/root/filter/etc/moira/storage-schemas.conf
	for service in "filter" "notifier" "api" "checker" "cli" ; do \
		tar -czvPf build/moira-$$service-${VERSION_RELEASE}.tar.gz -C build/root/$$service . ; \
	done

.PHONY: rpm
rpm: tar
	for service in $(SERVICES) ; do \
		fpm -t rpm \
			-s "tar" \
			--description "Moira $$service" \
			--vendor ${VENDOR} \
			--url ${URL} \
			--license ${LICENSE} \
			--name "moira-$$service" \
			--version "${VERSION_RELEASE}" \
			--iteration "1" \
			--config-files "/etc/moira/$$service.yml" \
			--after-install "./pkg/$$service/postinst" \
			-p build \
			build/moira-$$service-${VERSION_RELEASE}.tar.gz ; \
	done
	fpm -t rpm \
		-s "tar" \
		--description "Moira filter" \
		--vendor ${VENDOR} \
		--url ${URL} \
		--license ${LICENSE} \
		--name "moira-filter" \
		--version "${VERSION_RELEASE}" \
		--iteration "1" \
		--config-files "/etc/moira/filter.yml" \
		--config-files "/etc/moira/storage-schemas.conf" \
		--after-install "./pkg/filter/postinst" \
		-p build \
		build/moira-filter-${VERSION_RELEASE}.tar.gz

.PHONY: deb
deb: tar
	for service in $(SERVICES) ; do \
		fpm -t deb \
			-s "tar" \
			--description "Moira $$service" \
			--vendor ${VENDOR} \
			--url ${URL} \
			--license ${LICENSE} \
			--name "moira-$$service" \
			--version "${VERSION_RELEASE}" \
			--iteration "1" \
			--config-files "/etc/moira/$$service.yml" \
			--after-install "./pkg/$$service/postinst" \
			-p build \
			build/moira-$$service-${VERSION_RELEASE}.tar.gz ; \
	done
	fpm -t deb \
		-s "tar" \
		--description "Moira filter" \
		--vendor ${VENDOR} \
		--url ${URL} \
		--license ${LICENSE} \
		--name "moira-filter" \
		--version "${VERSION_RELEASE}" \
		--iteration "1" \
		--config-files "/etc/moira/filter.yml" \
		--config-files "/etc/moira/storage-schemas.conf" \
		--after-install "./pkg/filter/postinst" \
		-p build \
		build/moira-filter-${VERSION_RELEASE}.tar.gz

.PHONY: packages
packages: clean build tar rpm deb

.PHONY: docker_feature_images
docker_feature_images:
	for service in "filter" $(SERVICES) ; do \
		docker build --build-arg MoiraVersion=${VERSION_FEATURE} --build-arg GO_VERSION=${GO_VERSION} --build-arg GIT_COMMIT=${GIT_HASH} -f Dockerfile.$$service -t moira/$$service-${MARK_UNSTABLE}:${VERSION_FEATURE} . ; \
		docker push moira/$$service-${MARK_UNSTABLE}:${VERSION_FEATURE} ; \
	done

.PHONY: docker_nightly_images
docker_nightly_images:
	for service in "filter" $(SERVICES) ; do \
		docker build --build-arg MoiraVersion=${VERSION_NIGHTLY} --build-arg GO_VERSION=${GO_VERSION} --build-arg GIT_COMMIT=${GIT_HASH} -f Dockerfile.$$service -t moira/$$service-${MARK_NIGHTLY}:${VERSION_NIGHTLY} -t moira/$$service-${MARK_NIGHTLY}:latest . ; \
		docker push moira/$$service-${MARK_NIGHTLY}:${VERSION_NIGHTLY} ; \
		docker push moira/$$service-${MARK_NIGHTLY}:latest ; \
	done

.PHONY: docker_release_images
docker_release_images:
	for service in "filter" $(SERVICES) ; do \
		docker build --build-arg MoiraVersion=${VERSION_RELEASE} --build-arg GO_VERSION=${GO_VERSION} --build-arg GIT_COMMIT=${GIT_HASH} -f Dockerfile.$$service -t moira/$$service:${VERSION_RELEASE} -t moira/$$service:latest . ; \
		docker push moira/$$service:${VERSION_RELEASE} ; \
		docker push moira/$$service:latest ; \
	done
