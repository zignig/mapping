export CGO_ENABLED=0
build_id:=$(shell git rev-parse --short HEAD)
git_tag:=$(shell git describe --tags 2>/dev/null)

ifeq "$(git_tag)" ""
	version = $(build_id)
else
	version = $(git_tag)
endif

$(info build version=$(version))


prod:
	$(info build prod...)
	go-bindata -o src/mapping/bindata.go -prefix src/mapping/ src/mapping/asset/...
	gb build -f -ldflags '-w -s -X main.version=$(version)' 

debug:
	$(info build dev...)
	go-bindata -dev -o src/mapping/bindata.go -prefix src/mapping/ src/mapping/asset/...
	gb build -race -f -ldflags="-X main.build_type=debug -X main.version=$(version) -X main.rootDir=${CURDIR}/src/mapping"

