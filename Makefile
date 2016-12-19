# Path to gulp executable for building the client
GULP=./node_modules/.bin/gulp

# Version for tagging the releases
VERSION=$(shell git describe --abbrev=0 --tags)

# Differentiate between Unix and mingw builds
ifeq ($(OS), Windows_NT)
	BUILD_PATH="/.meguca_build/src/github.com/bakape"
	export GOPATH="/.meguca_build"
	export PKG_CONFIG_PATH:=$(PKG_CONFIG_PATH):/mingw64/lib/pkgconfig/
	export PKG_CONFIG_LIBDIR=/mingw64/lib/pkgconfig/
	export PATH:=$(PATH):/mingw64/bin/
	BINARY=meguca.exe
	ISWINDOWS=true
else
	BUILD_PATH="./.build/src/github.com/bakape"
	export GOPATH=$(shell pwd)/.build
	BINARY=meguca
	ISWINDOWS=false
endif

.PHONY: server client

# Build everything
all: server client

# Install NPM deps and build client
client:
	npm install
	$(GULP)

# Incrementaly rebuild the client for faster develepment builds. Only builds
# the ES6 version for modern browsers.
watch:
	$(GULP) -w

# Build server
server: server_deps
	go build -v -o $(BINARY)
ifeq ($(ISWINDOWS), true)
	cp /mingw64/bin/*.dll ./
endif

# Fecth all server dependancies. Dependacies are not updated automatically.
server_deps: build_dirs
	go list -f '{{.Deps}}' . \
		| tr "[" " " \
		| tr "]" " " \
		| xargs go list -e -f '{{if not .Standard}}{{.ImportPath}}{{end}}' \
		| grep -v 'github.com/bakape/meguca' \
		| xargs go get -v

# Fetch updates of both meguca and dependancies
update: build_dirs
	go get -u -v github.com/bakape/meguca
	npm update

# Creates the temporary directories for compiling
build_dirs:
ifeq ($(ISWINDOWS), true)
	rm -rf $(BUILD_PATH)
endif
	mkdir -p $(BUILD_PATH)
	ln -sfn "$(shell pwd)" $(BUILD_PATH)/meguca

# Removes compiled client files
client_clean:
	rm -rf www/js www/css/*.css www/css/maps www/lang

# Removes any build and dependancy directories
clean: client_clean
	rm -rf .build .ffmpeg node_modules .package \
		meguca-*.zip meguca-*.tar.xz meguca meguca.exe
ifeq ($(ISWINDOWS), true)
	rm -rf /.meguca_build *.dll
endif

# Also removes runtime use dirs
dist_clean: clean
	rm -rf images error.log

# Run all server tests
test: server_deps
	go test ./...
