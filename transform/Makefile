flags=-ldflags="-s -w"
# flags=-ldflags="-s -w -extldflags -static"
TAG := $(shell git tag | sed -e "s,v,," | sort -r | head -n 1)

all: golib build

golib:
	./get_golib.sh

build:
	CGO_ENABLED=0 go build ${flags}

test:
	go test -v .
