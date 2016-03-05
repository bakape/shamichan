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

install:
	if ! getent passwd meguca > /dev/null 2>&1; then \
		useradd -MrU meguca; \
	fi
	mkdir -p $(MEGUCA_VAR)/img/src
	mkdir -p $(MEGUCA_VAR)/img/thumb
	mkdir -p $(MEGUCA_VAR)/img/mid
	mkdir -p /etc/meguca/assets
	cp -n config/defaults.json /etc/meguca/config.json
	cp -r www /var/lib/meguca
	cp meguca /usr/bin

uninstall:
	if getent passwd meguca > /dev/null 2>&1; then \
		userdel meguca; \
	fi
	rm -r /var/lib/meguca/www
	rm /usr/bin/meguca

upgrade: uninstall install
