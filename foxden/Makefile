flags=-ldflags="-s -w"
# flags=-ldflags="-s -w -extldflags -static"
TAG := $(shell git tag | sed -e "s,v,," | sort -r | head -n 1)

all: golib build

golib:
	./get_golib.sh

build:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" cmd/version.go
endif
	CGO_ENABLED=0 go build ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" cmd/version.go
endif

build_all: golib build_darwin_amd64 build_darwin_arm64 build_amd64 build_arm64 build_power8 build_risc64 build_windows_amd64 build_windows_arm64

build_darwin_amd64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" cmd/version.go
endif
	CGO_ENABLED=0 GOOS=darwin go build ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" cmd/version.go
endif
	mv foxden foxden_darwin_amd64

build_darwin_arm64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" cmd/version.go
endif
	CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" cmd/version.go
endif
	mv foxden foxden_darwin_arm64

build_amd64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" cmd/version.go
endif
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" cmd/version.go
endif
	mv foxden foxden_linux_amd64

build_arm64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" cmd/version.go
endif
	CGO_ENABLED=0 go GOARCH=arm64 GOOS=linux build ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" cmd/version.go
endif
	mv foxden foxden_linux_arm64

build_power8:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" cmd/version.go
endif
	CGO_ENABLED=0 GOARCH=ppc64le GOOS=linux go build ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" cmd/version.go
endif
	mv foxden foxden_linux_power8

build_risc64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" cmd/version.go
endif
	CGO_ENABLED=0 GOARCH=riscv64 GOOS=linux go build ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" cmd/version.go
endif
	mv foxden foxden_linux_risc64

build_windows_amd64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" cmd/version.go
endif
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" cmd/version.go
endif
	mv foxden foxden_windows_amd64.exe

build_windows_arm64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" cmd/version.go
endif
	CGO_ENABLED=0 GOARCH=arm64 GOOS=windows go build ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" cmd/version.go
endif
	mv foxden foxden_windows_arm64.exe

test:
	go test -v .
