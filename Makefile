flags=-ldflags="-s -w"
TAG := $(shell git tag | sed -e "s,v,," | sort -r | head -n 1)

TOOLS := enc foxden validator migrate transform hostinfo gentoken foxden-benchmark migrate migratescans genprovenance

all: build

golib:
	./get_golib.sh

gorelease:
	goreleaser release --snapshot --clean

# --------------------------------------------------------------------
# Helper: build all tools for given OS and ARCH
# --------------------------------------------------------------------
define build_tools
	@for tool in $(TOOLS); do \
		echo "==> Building $$tool for $(1)/$(2)"; \
		( cd $$tool && CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) make ); \
	done
endef

# --------------------------------------------------------------------
# Helper: package built binaries into archive
# Only moves built binaries (not directories)
# --------------------------------------------------------------------
define package_tools
    mkdir -p tools
    EXT=
    if [ "$(1)" = "windows" ]; then EXT=".exe"; fi; \
    for tool in $(TOOLS); do \
        BIN="$$tool/$$tool$$EXT"; \
        if [ -f $$BIN ]; then \
            cp $$BIN tools/; \
        else \
            echo "Warning: missing $$BIN"; \
        fi; \
    done; \
    if [ "$(1)" = "windows" ]; then \
        zip -r tools_$(1)_$(2).zip tools; \
    else \
        tar cfz tools_$(1)_$(2).tar.gz tools; \
    fi; \
    rm -rf tools
endef

# --------------------------------------------------------------------
# Native build
# --------------------------------------------------------------------
build:
	@for tool in $(TOOLS); do \
		echo "==> Building $$tool"; \
		( cd $$tool && CGO_ENABLED=0 make ); \
	done

# --------------------------------------------------------------------
# Cross builds
# --------------------------------------------------------------------
build_darwin_amd64:
	$(call build_tools,darwin,amd64)
	$(call package_tools,darwin,amd64)

build_darwin_arm64:
	$(call build_tools,darwin,arm64)
	$(call package_tools,darwin,arm64)

build_linux_amd64:
	$(call build_tools,linux,amd64)
	$(call package_tools,linux,amd64)

build_linux_arm64:
	$(call build_tools,linux,arm64)
	$(call package_tools,linux,arm64)

build_linux_power8:
	$(call build_tools,linux,ppc64le)
	$(call package_tools,linux,power8)

build_windows_amd64:
	$(call build_tools,windows,amd64)
	$(call package_tools,windows,amd64)

build_windows_arm64:
	$(call build_tools,windows,arm64)
	$(call package_tools,windows,arm64)

# --------------------------------------------------------------------
# Aggregate targets
# --------------------------------------------------------------------
build_all: build_darwin_amd64 build_darwin_arm64 build_linux_amd64 build_linux_arm64 build_linux_power8 build_windows_amd64 build_windows_arm64 changes

changes:
	./changes.sh
	./last_changes.sh

test:
	echo "No tests so far"
# Example: go test -v -run TestFilesDB

