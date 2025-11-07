flags=-ldflags="-s -w"
# flags=-ldflags="-s -w -extldflags -static"

all: build

build:
	CGO_ENABLED=0 go build ${flags}

build_all: golib build_darwin_amd64 build_darwin_arm64 build_amd64 build_arm64 build_power8 build_risc64 build_windows_amd64 build_windows_arm64

build_darwin_amd64:
	CGO_ENABLED=0 GOOS=darwin go build ${flags}
	mv globuslink globuslink_darwin_amd64

build_darwin_arm64:
	CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build ${flags}
	mv globuslink globuslink_darwin_arm64

build_amd64:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build ${flags}
	mv globuslink globuslink_linux_amd64

build_arm64:
	CGO_ENABLED=0 go GOARCH=arm64 GOOS=linux build ${flags}
	mv globuslink globuslink_linux_arm64

build_power8:
	CGO_ENABLED=0 GOARCH=ppc64le GOOS=linux go build ${flags}
	mv globuslink globuslink_linux_power8

build_risc64:
	CGO_ENABLED=0 GOARCH=riscv64 GOOS=linux go build ${flags}
	mv globuslink globuslink_linux_risc64

build_windows_amd64:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build ${flags}
	mv globuslink globuslink_windows_amd64.exe

build_windows_arm64:
	CGO_ENABLED=0 GOARCH=arm64 GOOS=windows go build ${flags}
	mv globuslink globuslink_windows_arm64.exe

test:
	go test -v .
