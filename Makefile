GULP="./node_modules/.bin/gulp"
MEGUCA_VAR=/var/lib/meguca

ifeq ($(OS), Windows_NT)
	BUILD_PATH="/.meguca_build/src/github.com/bakape"
	export GOPATH="/.meguca_build"
	BINARY=meguca.exe
	ISWINDOWS=true
else
	BUILD_PATH="./.build/src/github.com/bakape"
	export GOPATH=$(shell pwd)/.build
	BINARY=meguca
	ISWINDOWS=false
endif

.PHONY: client server

all: server client

client:
	npm update
	$(GULP)
	$(GULP) es5

watch:
	$(GULP) -w

server: server_deps
	go build -o $(BINARY)

server_deps: build_dirs
	go list -f '{{.Deps}}' . \
		| tr "[" " " \
		| tr "]" " " \
		| xargs go list -e -f '{{if not .Standard}}{{.ImportPath}}{{end}}' \
		| grep -v 'github.com/bakape/meguca' \
		| xargs go get -v

build_dirs:
	if $(ISWINDOWS) = true; then \
		rm -rf $(BUILD_PATH); \
	fi
	mkdir -p $(BUILD_PATH)
	 ln -sf "$(shell pwd)" $(BUILD_PATH)

clean: client_clean
	rm -rf .build node_modules $(BINARY)

client_clean:
	rm -rf www/js www/css/*.css www/css/maps www/lang
	if $(ISWINDOWS) = true; then \
		rm -rf /.meguca_build; \
	fi

dist_clean: clean
	rm -rf img config/config.json assets error.log

init:
	mkdir -p img/src
	mkdir -p img/thumb
	mkdir -p img/mid
	mkdir -p assets
	cp -n config/defaults.json config/config.json

test: server_deps
	go get -v gopkg.in/check.v1
	go test -v ./...
