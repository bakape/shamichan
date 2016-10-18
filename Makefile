# Path to gulp executable for building the client
GULP=./node_modules/.bin/gulp

# Version for tagging the releases
VERSION=$(shell git describe --abbrev=0 --tags)

# Path to and target for the MXE cross environment for cross-compiling to
# win_amd64. Default value is the debian x86-static install path.
MXE_ROOT=/usr/lib/mxe/usr
MXE_TARGET=x86_64-w64-mingw32.static

# Root of statically compiled libraries for native static compilation
STATIC_ROOT=/usr/local

# Differentiate between Unix and mingw builds
ifeq ($(OS), Windows_NT)
	BUILD_PATH="/.meguca_build/src/github.com/bakape"
	export GOPATH="/.meguca_build"
	export PKG_CONFIG_PATH:=$(PKG_CONFIG_PATH):/mingw64/lib/pkgconfig/
	export PKG_CONFIG_LIBDIR=/mingw64/lib/pkgconfig/
	export PATH:=$(PATH):/mingw64/bin/
	BINARY=meguca.exe
	ISWINDOWS=true
	PACKAGE="meguca-$(VERSION)_windows_$(PROCESSOR_ARCHITECTURE).zip"
else
	BUILD_PATH="./.build/src/github.com/bakape"
	export GOPATH=$(shell pwd)/.build
	BINARY=meguca
	ISWINDOWS=false
	OS_LOWER=$(shell echo `uname -s` | tr A-Z a-z)
	ARCH=$(shell uname -p)
	PACKAGE="meguca-$(VERSION)_$(OS_LOWER)_$(ARCH).tar.xz"
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

# Build server, but link all cgo deps statically
server_static:
	PKG_CONFIG_LIBDIR=$(STATIC_ROOT)/lib/pkgconfig \
	PKG_CONFIG_PATH=$(STATIC_ROOT)/lib/pkgconfig \
	go build -v -a -o $(BINARY) --ldflags '-extldflags "-static"'

# Fecth all server dependancies. Dependacies are not updated automatically.
server_deps: build_dirs
	go list -f '{{.Deps}}' . \
		| tr "[" " " \
		| tr "]" " " \
		| xargs go list -e -f '{{if not .Standard}}{{.ImportPath}}{{end}}' \
		| grep -v 'github.com/bakape/meguca' \
		| xargs go get -v

# Fetch updates of both meguca and dependancies
update:
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
	rm -rf .build .ffmpeg node_modules $(BINARY) .package
ifeq ($(ISWINDOWS), true)
	rm -rf /.meguca_build *.dll
endif

# Also removes runtime use dirs
dist_clean: clean
	rm -rf images error.log

# Run all server tests
test: server_deps
	go test ./...

# Build ffmpeg for integration testing with Travis.cl. We need these, because
# their servers are still running trusty.
travis_build_ffmpeg:
	apt-get install -y libvpx-dev libmp3lame-dev libopus-dev libvorbis-dev \
		libx264-dev libtheora-dev git build-essential yasm
ifeq ("$(wildcard .ffmpeg/ffmpeg)", "")
	git clone --depth 1 -b release/3.0 https://github.com/FFmpeg/FFmpeg.git \
		.ffmpeg
	cd .ffmpeg; \
	./configure --enable-libmp3lame --enable-libx264 --enable-libvpx \
		--enable-libvorbis --enable-libopus --enable-libtheora --enable-gpl
	$(MAKE) -C .ffmpeg
endif
	$(MAKE) -C .ffmpeg install

# Generate binary packages for distribution
package: server_static client package_copy
	rm -rf .package/meguca.exe
	cp $(BINARY) .package/
ifeq ($(ISWINDOWS), true)
	cp *.dll .package/
	cd .package; zip -rq ../$(PACKAGE) .
else
	cd .package; tar cfpJ ../$(PACKAGE) *
endif

# Copy generated package contents for archiving
package_copy:
	rm -rf .package
	mkdir -p .package/templates .package/images/src .package/images/thumb
	cp -r docs scripts www CHANGELOG.md README.md LICENSE .package/
	cp -r templates/*.html .package/templates/

# Cross-compile from Unix into a Windows_amd64 static binary
# Needs Go checkout dfbbe06a205e7048a8541c4c97b250c24c40db96 or later. At the
# moment of writing this change is not released yet. Should probably make it
# into Go 1.7.1.
# Depends on:
# 	mxe-x86-64-w64-mingw32.static-gcc
# 	mxe-x86-64-w64-mingw32.static-libidn
# 	mxe-x86-64-w64-mingw32.static-ffmpeg
cross_compile_win_amd64:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
	CC=$(MXE_ROOT)/bin/$(MXE_TARGET)-gcc \
	PKG_CONFIG=$(MXE_ROOT)/bin/$(MXE_TARGET)-pkg-config \
	PKG_CONFIG_LIBDIR=$(MXE_ROOT)/$(MXE_TARGET)/lib/pkgconfig \
	PKG_CONFIG_PATH=$(MXE_ROOT)/$(MXE_TARGET)/lib/pkgconfig \
	go build -v -a -o meguca.exe --ldflags '-extldflags "-static"'

# Zip the cross-compiled contents into an archive
cross_package_win_amd64: cross_compile_win_amd64 client package_copy
	rm -rf .package/meguca
	cp meguca.exe .package/
	cd .package; zip -rq ../meguca-$(VERSION)_windows_x86_64.zip .
