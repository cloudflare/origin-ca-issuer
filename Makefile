.DEFAULT_GOAL := binaries

KERNEL := $(shell uname -s)
GOTESTSUM := $(shell command -v gotestsum 2> /dev/null)

DIB ?= docker
IMAGE_ROOT ?= localhost/origin-ca-issuer
IMAGE_VERSION ?= $(shell git log -1 --pretty=format:%cd-%h --date short HEAD)
VERSION := $(shell git describe --tags --always --dirty=-dev)
# Build docker images for the native arch, but allow overriding in the environment for local development
PLATFORM ?= local

# Bind mount $SSL_CERT_FILE (or default) to build container if the file exists.
SSL_CERT_FILE ?= /etc/ssl/certs/ca-certificates.crt
ifneq (,$(wildcard ${SSL_CERT_FILE}))
SECRETS = --secret id=certificates,src=${SSL_CERT_FILE}
endif

# When compiling for Linux enable Security's recommend hardening to satisfy `checksec' checks.
# Unfortunately, most of these flags aren't portable to other operating systems.
ifeq (${KERNEL},Linux)
	CGO_ENABLED ?= 1
	CPPFLAGS ?= -D_FORTIFY_SOURCE=2 -fstack-protector-all
	CFLAGS ?= -O2 -pipe -fno-plt
	CXXFLAGS ?= -O2 -pipe -fno-plt
	LDFLAGS ?= -Wl,-O1,-sort-common,-as-needed,-z,relro,-z,now
	GO_LDFLAGS ?= -linkmode=external
	GOFLAGS ?= -buildmode=pie
endif

GO_LDFLAGS += -w -s -X main.version=${VERSION}
GOFLAGS += -v

export CGO_ENABLED
export CGO_CPPFLAGS ?= ${CPPFLAGS}
export CGO_CFLAGS ?= ${CFLAGS}
export CGO_CXXFLAGS ?= ${CXXFLAGS}
export CGO_LDFLAGS ?= ${LDFLAGS}

CMDS := $(shell find cmd -mindepth 1 -maxdepth 1 -type d | awk -F '/' '{ print $$NF }' )
IMAGES := $(shell find cmd -mindepth 1 -type f -name Dockerfile | awk -F '/' '{ print $$2 }')

define make-go-target
.PHONY: bin/$1
bin/$1:
	go build ${GOFLAGS} -o $$@ -ldflags "${GO_LDFLAGS}" ./cmd/$1
endef

define make-dib-targets
.PHONY: images/$1
images/$1:
	${DIB} buildx build --platform "$(PLATFORM)" ${SECRETS} -f cmd/$1/Dockerfile -t "${IMAGE_ROOT}/$1:${IMAGE_VERSION}" .

.PHONY: push/images/$1
push/images/$1:
	${DIB} push "${IMAGE_ROOT}/$1:${IMAGE_VERSION}"
endef

$(foreach element,$(CMDS), $(eval $(call make-go-target,$(element))))
$(foreach element,$(IMAGES), $(eval $(call make-dib-targets,$(element))))

.PHONY: binaries
binaries: $(CMDS:%=bin/%)

.PHONY: images
images: $(IMAGES:%=images/%)

.PHONY: push-images
push-images: $(IMAGES:%=push/images/%)

.PHONY: clean
clean:
	rm -rf bin

.PHONY: test
test:
ifdef GOTESTSUM
	"${GOTESTSUM}" -- -count 1 ./...
else
	go test -cover -count 1 ./...
endif

.PHONY: lint
lint:
	staticcheck -tags suite ./...
