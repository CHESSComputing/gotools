flags=-ldflags="-s -w"
# flags=-ldflags="-s -w -extldflags -static"
TAG := $(shell git tag | sed -e "s,v,," | sort -r | head -n 1)

all: build

build:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" cmd/version.go
endif
	CGO_ENABLED=0 go build ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" cmd/version.go
endif

test:
	go test -v .
