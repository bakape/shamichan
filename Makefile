GULP="./node_modules/.bin/gulp"
LINKED_PATH="./.build/src/github.com/bakape"
MEGUCA_VAR=/var/lib/meguca

export GOPATH=$(shell pwd)/.build

.PHONY: client server

all: server client

client:
	npm update
	$(GULP)
	$(GULP) es5

watch:
	$(GULP) -w

server: server_deps
	go build -o meguca

server_deps: build_dirs
	go list -f '{{.Deps}}' . \
		| tr "[" " " \
		| tr "]" " " \
		| xargs go list -e -f '{{if not .Standard}}{{.ImportPath}}{{end}}' \
		| grep -v 'github.com/bakape/meguca' \
		| xargs go get -v

build_dirs:
	mkdir -p $(LINKED_PATH)
	ln -sf "$(shell pwd)" $(LINKED_PATH)

clean: client_clean
	rm -rf .build node_modules meguca

client_clean:
	rm -rf www/js www/css/*.css www/lang

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
