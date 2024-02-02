flags=-ldflags="-s -w"
TAG := $(shell git tag | sed -e "s,v,," | sort -r | head -n 1)

all: build

golib:
	./get_golib.sh

gorelease:
	goreleaser release --snapshot --clean

build:
	cd enc; go build; cd -
	cd foxden; 	make; cd -
	cd validator; go build; cd -

build_all: build_darwin_amd64 build_darwin_arm64 build_linux_amd64 build_linux_arm64 build_linux_power8 build_windows_amd64 build_windows_arm64

build_darwin_amd64:
	cd enc; GOOS=darwin go build; cd -
	cd foxden; 	GOOS=darwin make; cd -
	cd validator; GOOS=darwin go build; cd -
	tar cfz tools_darwin_amd64.tar.gz enc/enc foxden/foxden validator/validator

build_darwin_arm64:
	cd enc; GOARCH=arm64 GOOS=darwin go build; cd -
	cd foxden; 	GOARCH=arm64 GOOS=darwin make; cd -
	cd validator; GOARCH=arm64 GOOS=darwin go build; cd -
	tar cfz tools_darwin_arm64.tar.gz enc/enc foxden/foxden validator/validator

build_linux_amd64:
	cd enc; GOOS=linux go build; cd -
	cd foxden; 	GOOS=linux make; cd -
	cd validator; GOOS=linux go build; cd -
	tar cfz tools_linux_amd64.tar.gz enc/enc foxden/foxden validator/validator

build_linux_power8:
	cd enc; GOARCH=ppc64le GOOS=linux go build; cd -
	cd foxden; 	GOARCH=ppc64le GOOS=linux make; cd -
	cd validator; GOARCH=ppc64le GOOS=linux go build; cd -
	tar cfz tools_linux_power8.tar.gz enc/enc foxden/foxden validator/validator

build_linux_arm64:
	cd enc; GOARCH=arm64 GOOS=darwin go build; cd -
	cd foxden; 	GOARCH=arm64 GOOS=darwin make; cd -
	cd validator; GOARCH=arm64 GOOS=darwin go build; cd -
	tar cfz tools_linux_arm64.tar.gz enc/enc foxden/foxden validator/validator

build_windows_amd64:
	cd enc; GOOS=windows go build; cd -
	cd foxden;	GOOS=windows make; cd -
	cd validator; GOOS=windows go build; cd -
	tar cfz tools_windows_amd64.tar.gz enc/enc* foxden/foxden* validator/validator*

build_windows_arm64:
	cd enc; GOARCH=arm64 GOOS=windows go build; cd -
	cd foxden; 	GOARCH=arm64 GOOS=windows make; cd -
	cd validator; GOARCH=arm64 GOOS=windows go build; cd -
	tar cfz tools_windows_arm64.tar.gz enc/enc* foxden/foxden* validator/validator*

# here is an example for execution of individual test
# go test -v -run TestFilesDB
