#GOPATH:=$(PWD):${GOPATH}
#export GOPATH
flags=-ldflags="-s -w"
# flags=-ldflags="-s -w -extldflags -static"

# latest version tag without the leading "v" (version sort: 1.10 > 1.9)
TAG := $(shell git tag --sort=-v:refname 2>/dev/null | head -n 1 | sed -e 's,^v,,')
# value substituted for {{VERSION}} in main.go (unchanged when there is no tag)
VER := $(if $(TAG),$(TAG),{{VERSION}})

HOSTOS   := $(shell go env GOOS)
HOSTARCH := $(shell go env GOARCH)

# every cross-platform binary produced by build_all (names are kept as before)
PLATFORM_TARGETS := \
	build_darwin_amd64 build_darwin_arm64 \
	build_amd64 build_amd64_static build_arm64 build_arm build_386 \
	build_power8 build_s390x build_riscv64 \
	build_freebsd_amd64 \
	build_windows_amd64 build_windows_arm64
OUTPUTS := foxden foxden_darwin_amd64 foxden_darwin_arm64 \
	foxden_amd64 foxden_amd64_static foxden_arm64 foxden_arm foxden_386 \
	foxden_power8 foxden_s390x foxden_riscv64 foxden_freebsd_amd64 \
	foxden_amd64.exe foxden_arm64.exe

.PHONY: all golib gorelease build build_all $(PLATFORM_TARGETS) \
	    install clean changes test test_code
# main.go is temporarily rewritten while building, so builds must not overlap
.NOTPARALLEL:

all: golib build

golib:
	./get_golib.sh

gorelease:
	goreleaser release --snapshot --clean

# $(call xbuild,GOOS,GOARCH,OUTPUT[,ENV_VARS][,EXTRA_GO_FLAGS])
# - injects the version into main.go from a backup copy
# - always restores main.go, even if the build fails
# - exits with the exit code of "go build"
define xbuild
	@echo "==> building $(3) for $(1)/$(2) architecture"
	@rm -f $(3)
	@cp main.go main.go.orig; \
	sed -e "s,{{VERSION}},$(VER),g" main.go.orig > main.go; \
	$(4) GOOS=$(1) GOARCH=$(2) go build $(5) $(flags) -o $(3); rc=$$?; \
	mv main.go.orig main.go; \
	exit $$rc
endef

# native build for the current machine
build:
	$(call xbuild,$(HOSTOS),$(HOSTARCH),foxden)

build_all: golib $(PLATFORM_TARGETS) changes

# macOS
build_darwin_amd64:
	$(call xbuild,darwin,amd64,foxden_darwin_amd64)
build_darwin_arm64:
	$(call xbuild,darwin,arm64,foxden_darwin_arm64)

# Linux
build_amd64:
	$(call xbuild,linux,amd64,foxden_amd64)
build_amd64_static:
	$(call xbuild,linux,amd64,foxden_amd64_static,CGO_ENABLED=0,-tags static)
build_arm64:
	$(call xbuild,linux,arm64,foxden_arm64)
build_arm:
	$(call xbuild,linux,arm,foxden_arm,GOARM=7)
build_386:
	$(call xbuild,linux,386,foxden_386)
build_power8:
	$(call xbuild,linux,ppc64le,foxden_power8)
build_s390x:
	$(call xbuild,linux,s390x,foxden_s390x)
build_riscv64:
	$(call xbuild,linux,riscv64,foxden_riscv64)

# FreeBSD
build_freebsd_amd64:
	$(call xbuild,freebsd,amd64,foxden_freebsd_amd64)

# Windows
build_windows_amd64:
	$(call xbuild,windows,amd64,foxden_amd64.exe)
build_windows_arm64:
	$(call xbuild,windows,arm64,foxden_arm64.exe)

install:
	go install

clean:
	go clean; rm -rf pkg; rm -f $(OUTPUTS) main.go.orig

changes:
	./changes.sh
	./last_changes.sh

test : test_code

test_code:
	touch ~/.foxden.yaml
	go test -test.v .

# here is an example for execution of individual test
# go test -v -run TestFilesDB
