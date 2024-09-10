flags=-ldflags="-s -w"
TAG := $(shell git tag | sed -e "s,v,," | sort -r | head -n 1)

all: build

golib:
	./get_golib.sh

gorelease:
	goreleaser release --snapshot --clean

build:
	cd enc; CGO_ENABLED=0 go build; cd -
	cd foxden; CGO_ENABLED=0 make; cd -
	cd validator; CGO_ENABLED=0 go build; cd -
	cd migrate; CGO_ENABLED=0 go build; cd -
	cd transform; CGO_ENABLED=0 go build; cd -

build_all: build_darwin_amd64 build_darwin_arm64 build_linux_amd64 build_linux_arm64 build_linux_power8 build_windows_amd64 build_windows_arm64 changes

build_darwin_amd64:
	cd enc; CGO_ENABLED=0 GOOS=darwin go build; cd -
	cd foxden; CGO_ENABLED=0 GOOS=darwin make; cd -
	cd validator; CGO_ENABLED=0 GOOS=darwin go build; cd -
	cd migrate; CGO_ENABLED=0 GOOS=darwin go build; cd -
	cd transform; CGO_ENABLED=0 GOOS=darwin go build; cd -
	cd hostinfo; CGO_ENABLED=0 GOOS=darwin go build; cd -
	mkdir tools
	mv enc/enc foxden/foxden validator/validator migrate/migrate transform/transform hostinfo/hostinfo tools
	tar cfz tools_darwin_amd64.tar.gz tools
	rm -rf tools

build_darwin_arm64:
	cd enc; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build; cd -
	cd foxden; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin make; cd -
	cd validator; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build; cd -
	cd migrate; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build; cd -
	cd transform; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build; cd -
	cd hostinfo; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build; cd -
	mkdir tools
	mv enc/enc foxden/foxden validator/validator migrate/migrate transform/transform hostinfo/hostinfo tools
	tar cfz tools_darwin_arm64.tar.gz tools
	rm -rf tools

build_linux_amd64:
	cd enc; CGO_ENABLED=0 GOOS=linux go build; cd -
	cd foxden; CGO_ENABLED=0 GOOS=linux make; cd -
	cd validator; CGO_ENABLED=0 GOOS=linux go build; cd -
	cd migrate; CGO_ENABLED=0 GOOS=linux go build; cd -
	cd transform; CGO_ENABLED=0 GOOS=linux go build; cd -
	cd hostinfo; CGO_ENABLED=0 GOOS=linux go build; cd -
	mkdir tools
	mv enc/enc foxden/foxden validator/validator migrate/migrate transform/transform hostinfo tools
	tar cfz tools_linux_amd64.tar.gz tools
	rm -rf tools

build_linux_power8:
	cd enc; CGO_ENABLED=0 GOARCH=ppc64le GOOS=linux go build; cd -
	cd foxden; CGO_ENABLED=0 GOARCH=ppc64le GOOS=linux make; cd -
	cd validator; CGO_ENABLED=0 GOARCH=ppc64le GOOS=linux go build; cd -
	cd migrate; CGO_ENABLED=0 GOARCH=ppc64le GOOS=linux go build; cd -
	cd transform; CGO_ENABLED=0 GOARCH=ppc64le GOOS=linux go build; cd -
	cd hostinfo; CGO_ENABLED=0 GOARCH=ppc64le GOOS=linux go build; cd -
	mkdir tools
	mv enc/enc foxden/foxden validator/validator migrate/migrate transform/transform hostinfo tools
	tar cfz tools_linux_power8.tar.gz tools
	rm -rf tools

build_linux_arm64:
	cd enc; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build; cd -
	cd foxden; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin make; cd -
	cd validator; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build; cd -
	cd migrate; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build; cd -
	cd transform; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build; cd -
	cd hostinfo; CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build; cd -
	mkdir tools
	mv enc/enc foxden/foxden validator/validator migrate/migrate transform/transform hostinfo tools
	tar cfz tools_linux_arm64.tar.gz tools
	rm -rf tools

build_windows_amd64:
	cd enc; CGO_ENABLED=0 GOOS=windows go build; cd -
	cd foxden; CGO_ENABLED=0 GOOS=windows make; cd -
	cd validator; CGO_ENABLED=0 GOOS=windows go build; cd -
	cd migrate; CGO_ENABLED=0 GOOS=windows go build; cd -
	cd transform; CGO_ENABLED=0 GOOS=windows go build; cd -
	cd hostinfo; CGO_ENABLED=0 GOOS=windows go build; cd -
	mkdir tools
	mv enc/enc.exe foxden/foxden.exe validator/validator.exe migrate/migrate.exe transform/transform.exe hostinfo/hostinfo.exe tools
	zip -r tools_windows_amd64.zip tools
	rm -rf tools

build_windows_arm64:
	cd enc; CGO_ENABLED=0 GOARCH=arm64 GOOS=windows go build; cd -
	cd foxden; CGO_ENABLED=0 GOARCH=arm64 GOOS=windows make; cd -
	cd validator; CGO_ENABLED=0 GOARCH=arm64 GOOS=windows go build; cd -
	cd migrate; CGO_ENABLED=0 GOARCH=arm64 GOOS=windows go build; cd -
	cd transform; CGO_ENABLED=0 GOARCH=arm64 GOOS=windows go build; cd -
	cd hostinfo; CGO_ENABLED=0 GOARCH=arm64 GOOS=windows go build; cd -
	mkdir tools
	mv enc/enc.exe foxden/foxden.exe validator/validator.exe migrate/migrate.exe transform/transform.exe hostinfo/hostinfo.exe tools
	zip -r tools_windows_arm64.zip tools
	rm -rf tools

changes:
	./changes.sh
	./last_changes.sh

test:
	echo "No tests so far"
# here is an example for execution of individual test
# go test -v -run TestFilesDB
