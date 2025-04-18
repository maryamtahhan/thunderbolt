export GOPROXY=https://proxy.golang.org

APPARMORTAG := $(shell hack/apparmor_tag.sh)
STORAGETAGS := exclude_graphdriver_devicemapper $(shell ./btrfs_tag.sh) $(shell ./btrfs_installed_tag.sh) $(shell ./hack/libsubid_tag.sh)
SECURITYTAGS ?= seccomp $(APPARMORTAG)
TAGS ?= $(SECURITYTAGS) $(STORAGETAGS) $(shell ./hack/systemd_tag.sh)
ifeq ($(shell uname -s),FreeBSD)
# FreeBSD needs CNI until netavark is supported
TAGS += cni
endif
BUILDTAGS += $(TAGS)
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
BASHINSTALLDIR = $(PREFIX)/share/bash-completion/completions
BUILDFLAGS := -tags "$(BUILDTAGS)"
BUILDAH := buildah
SELINUXOPT ?= $(shell test -x /usr/sbin/selinuxenabled && selinuxenabled && echo -Z)
SELINUXTYPE=container_runtime_exec_t
AS ?= as
STRIP ?= strip

GO := go
GO_LDFLAGS := $(shell if $(GO) version|grep -q gccgo; then echo "-gccgoflags"; else echo "-ldflags"; fi)
GO_GCFLAGS := $(shell if $(GO) version|grep -q gccgo; then echo "-gccgoflags"; else echo "-gcflags"; fi)
export GO_BUILD=$(GO) build
export GO_TEST=$(GO) test
RACEFLAGS := $(shell $(GO_TEST) -race ./pkg/dummy > /dev/null 2>&1 && echo -race)

COMMIT_NO ?= $(shell git rev-parse HEAD 2> /dev/null || true)
GIT_COMMIT ?= $(if $(shell git status --porcelain --untracked-files=no),${COMMIT_NO}-dirty,${COMMIT_NO})
SOURCE_DATE_EPOCH ?= $(if $(shell date +%s),$(shell date +%s),$(error "date failed"))
STATIC_STORAGETAGS = "containers_image_openpgp $(STORAGE_TAGS)"

# we get GNU make 3.x in MacOS build envs, which wants # to be escaped in
# strings, while the 4.x we have on Linux doesn't. this is the documented
# workaround
COMMENT := \#
CNI_COMMIT := $(shell sed -n 's;^$(COMMENT) github.com/containernetworking/cni \([^ \n]*\).*$$;\1;p' vendor/modules.txt)
RUNC_COMMIT := $(shell sed -n 's;^$(COMMENT) github.com/opencontainers/runc \([^ \n]*\).*$$;\1;p' vendor/modules.txt)
LIBSECCOMP_COMMIT := release-2.3

EXTRA_LDFLAGS ?=
BUILDAH_LDFLAGS := $(GO_LDFLAGS) '-X main.GitCommit=$(GIT_COMMIT) -X main.buildInfo=$(SOURCE_DATE_EPOCH) -X main.cniVersion=$(CNI_COMMIT) $(EXTRA_LDFLAGS)'
SOURCES=*.go imagebuildah/*.go bind/*.go chroot/*.go copier/*.go define/*.go docker/*.go internal/config/*.go internal/mkcw/*.go internal/mkcw/types/*.go internal/open/*.go internal/parse/*.go internal/sbom/*.go internal/source/*.go internal/tmpdir/*.go internal/*.go internal/util/*.go internal/volumes/*.go manifests/*.go pkg/binfmt/*.go pkg/blobcache/*.go pkg/chrootuser/*.go pkg/cli/*.go pkg/completion/*.go pkg/formats/*.go pkg/jail/*.go pkg/overlay/*.go pkg/parse/*.go pkg/rusage/*.go pkg/sshagent/*.go pkg/umask/*.go pkg/util/*.go pkg/volumes/*.go util/*.go

LINTFLAGS ?=

ifeq ($(BUILDDEBUG), 1)
  override GOGCFLAGS += -N -l
endif

#   make all BUILDDEBUG=1
#     Note: Uses the -N -l go compiler options to disable compiler optimizations
#           and inlining. Using these build options allows you to subsequently
#           use source debugging tools like delve.
all: bin/buildah bin/imgtype bin/copy bin/inet bin/tutorial docs

# Update nix/nixpkgs.json its latest stable commit
.PHONY: nixpkgs
nixpkgs:
	@nix run \
		-f channel:nixos-20.09 nix-prefetch-git \
		-c nix-prefetch-git \
		--no-deepClone \
		https://github.com/nixos/nixpkgs refs/heads/nixos-20.09 > nix/nixpkgs.json

# Build statically linked binary
.PHONY: static
static:
	@nix build -f nix/
	mkdir -p ./bin
	cp -rfp ./result/bin/* ./bin/

bin/buildah: $(SOURCES) cmd/buildah/*.go internal/mkcw/embed/entrypoint_amd64.gz
	$(GO_BUILD) $(BUILDAH_LDFLAGS) $(GO_GCFLAGS) "$(GOGCFLAGS)" -o $@ $(BUILDFLAGS) ./cmd/buildah
	test -z "${SELINUXOPT}" || chcon --verbose -t $(SELINUXTYPE) $@

ifneq ($(shell $(AS) --version | grep x86_64),)
internal/mkcw/embed/entrypoint_amd64.gz: internal/mkcw/embed/entrypoint_amd64
	gzip -k9nf $^

internal/mkcw/embed/entrypoint_amd64: internal/mkcw/embed/entrypoint_amd64.s
	$(AS) -o $(patsubst %.s,%.o,$^) $^
	$(LD) -o $@ $(patsubst %.s,%.o,$^)
	$(STRIP) $@
endif


.PHONY: buildah
buildah: bin/buildah

ALL_CROSS_TARGETS := $(addprefix bin/buildah.,$(subst /,.,$(shell $(GO) tool dist list)))
LINUX_CROSS_TARGETS := $(filter-out %.loong64,$(filter bin/buildah.linux.%,$(ALL_CROSS_TARGETS)))
DARWIN_CROSS_TARGETS := $(filter bin/buildah.darwin.%,$(ALL_CROSS_TARGETS))
WINDOWS_CROSS_TARGETS := $(addsuffix .exe,$(filter bin/buildah.windows.%,$(ALL_CROSS_TARGETS)))
FREEBSD_CROSS_TARGETS := $(filter bin/buildah.freebsd.%,$(ALL_CROSS_TARGETS))
.PHONY: cross
cross: $(LINUX_CROSS_TARGETS) $(DARWIN_CROSS_TARGETS) $(WINDOWS_CROSS_TARGETS) $(FREEBSD_CROSS_TARGETS)

bin/buildah.%:
	mkdir -p ./bin
	GOOS=$(word 2,$(subst ., ,$@)) GOARCH=$(word 3,$(subst ., ,$@)) $(GO_BUILD) $(BUILDAH_LDFLAGS) -o $@ -tags "containers_image_openpgp" ./cmd/buildah

bin/imgtype: $(SOURCES) tests/imgtype/imgtype.go
	$(GO_BUILD) $(BUILDAH_LDFLAGS) -o $@ $(BUILDFLAGS) ./tests/imgtype/imgtype.go

bin/copy: $(SOURCES) tests/copy/copy.go
	$(GO_BUILD) $(BUILDAH_LDFLAGS) -o $@ $(BUILDFLAGS) ./tests/copy/copy.go

bin/tutorial: $(SOURCES) tests/tutorial/tutorial.go
	$(GO_BUILD) $(BUILDAH_LDFLAGS) -o $@ $(BUILDFLAGS) ./tests/tutorial/tutorial.go

bin/inet: tests/inet/inet.go
	$(GO_BUILD) $(BUILDAH_LDFLAGS) -o $@ $(BUILDFLAGS) ./tests/inet/inet.go

.PHONY: clean
clean:
	$(RM) -r bin tests/testreport/testreport
	$(MAKE) -C docs clean

.PHONY: docs
docs: install.tools ## build the docs on the host
	$(MAKE) -C docs

# For vendoring to work right, the checkout directory must be such that our top
# level is at $GOPATH/src/github.com/containers/buildah.
.PHONY: gopath
gopath:
	test $(shell pwd) = $(shell cd ../../../../src/github.com/containers/buildah ; pwd)

codespell:
	codespell -S Makefile,buildah.spec.rpkg,AUTHORS,bin,vendor,.git,go.mod,go.sum,CHANGELOG.md,changelog.txt,seccomp.json,.cirrus.yml,"*.xz,*.gz,*.tar,*.tgz,*ico,*.png,*.1,*.5,*.orig,*.rej" -L secon,passt,bu,uint,iff,od,erro -w

.PHONY: validate
validate: install.tools
	./tests/validate/whitespace.sh
	./hack/xref-helpmsgs-manpages
	./tests/validate/pr-should-include-tests
	./hack/makefile_sources

.PHONY: install.tools
install.tools:
	$(MAKE) -C tests/tools

.PHONY: runc
runc: gopath
	rm -rf ../../opencontainers/runc
	git clone https://github.com/opencontainers/runc ../../opencontainers/runc
	cd ../../opencontainers/runc && git checkout $(RUNC_COMMIT) && $(GO) build -tags "$(STORAGETAGS) $(SECURITYTAGS)"
	ln -sf ../../opencontainers/runc/runc

.PHONY: install.libseccomp.sudo
install.libseccomp.sudo: gopath
	rm -rf ../../seccomp/libseccomp
	git clone https://github.com/seccomp/libseccomp ../../seccomp/libseccomp
	cd ../../seccomp/libseccomp && git checkout $(LIBSECCOMP_COMMIT) && ./autogen.sh && ./configure --prefix=/usr && make all && sudo make install

.PHONY: install.cni.sudo
install.cni.sudo: gopath
	rm -rf ../../containernetworking/plugins
	git clone https://github.com/containernetworking/plugins ../../containernetworking/plugins
	cd ../../containernetworking/plugins && ./build_linux.sh && sudo install -D -v -m755 -t /opt/cni/bin/ bin/*

.PHONY: install
install:
	install -d -m 755 $(DESTDIR)/$(BINDIR)
	install -m 755 bin/buildah $(DESTDIR)/$(BINDIR)/buildah
	$(MAKE) -C docs install

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)/$(BINDIR)/buildah
	rm -f $(PREFIX)/share/man/man1/buildah*.1
	rm -f $(DESTDIR)/$(BASHINSTALLDIR)/buildah

.PHONY: install.completions
install.completions:
	install -m 755 -d $(DESTDIR)/$(BASHINSTALLDIR)
	install -m 644 contrib/completions/bash/buildah $(DESTDIR)/$(BASHINSTALLDIR)/buildah

.PHONY: install.runc
install.runc:
	install -m 755 ../../opencontainers/runc/runc $(DESTDIR)/$(BINDIR)/

.PHONY: test-conformance
test-conformance:
	$(GO_TEST) -v -tags "$(STORAGETAGS) $(SECURITYTAGS)" -cover -timeout 60m ./tests/conformance

.PHONY: test-integration
test-integration: install.tools
	cd tests; ./test_runner.sh

tests/testreport/testreport: tests/testreport/testreport.go
	$(GO_BUILD) $(GO_LDFLAGS) "-linkmode external -extldflags -static" -tags "$(STORAGETAGS) $(SECURITYTAGS)" -o tests/testreport/testreport ./tests/testreport/testreport.go

.PHONY: test-unit
test-unit: tests/testreport/testreport
	$(GO_TEST) -v -tags "$(STORAGETAGS) $(SECURITYTAGS)" -cover $(RACEFLAGS) $(shell $(GO) list ./... | grep -v vendor | grep -v tests | grep -v cmd | grep -v chroot | grep -v copier) -timeout 45m
	$(GO_TEST) -v -tags "$(STORAGETAGS) $(SECURITYTAGS)"        $(RACEFLAGS) ./chroot ./copier -timeout 60m
	tmp=$(shell mktemp -d) ; \
	mkdir -p $$tmp/root $$tmp/runroot; \
	$(GO_TEST) -v -tags "$(STORAGETAGS) $(SECURITYTAGS)" -cover $(RACEFLAGS) ./cmd/buildah -args --root $$tmp/root --runroot $$tmp/runroot --storage-driver vfs --signature-policy $(shell pwd)/tests/policy.json --registries-conf $(shell pwd)/tests/registries.conf

vendor-in-container:
	if test -d `go env GOCACHE` && test -w `go env GOCACHE` ; then \
		podman run --privileged --rm --env HOME=/root -v `go env GOCACHE`:/root/.cache/go-build --env GOCACHE=/root/.cache/go-build -v `pwd`:/src -w /src docker.io/library/golang:1.21 make vendor ; \
	else \
		podman run --privileged --rm --env HOME=/root -v `pwd`:/src -w /src docker.io/library/golang:1.21 make vendor ; \
	fi

.PHONY: vendor
vendor:
	$(GO) mod tidy
	$(GO) mod vendor
	$(GO) mod verify
	if test -n "$(strip $(shell go env GOTOOLCHAIN))"; then go mod edit -toolchain none ; fi

.PHONY: lint
lint: install.tools
	./tests/tools/build/golangci-lint run $(LINTFLAGS)

# CAUTION: This is not a replacement for RPMs provided by your distro.
# Only intended to build and test the latest unreleased changes.
.PHONY: rpm
rpm:  ## Build rpm packages
	$(MAKE) -C rpm

# Remember that rpms install exec to /usr/bin/buildah while a `make install`
# installs them to /usr/local/bin/buildah which is likely before. Always use
# a full path to test installed buildah or you risk to call another executable.
.PHONY: rpm-install
rpm-install: package  ## Install rpm packages
	$(call err_if_empty,PKG_MANAGER) -y install rpm/RPMS/*/*.rpm
	/usr/bin/buildah version
