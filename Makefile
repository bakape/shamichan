export node_bins=$(PWD)/node_modules/.bin
export uglifyjs=$(node_bins)/uglifyjs
export gulp=$(node_bins)/gulp

# Differentiate between Unix and mingw builds
ifeq ($(OS), Windows_NT)
	build_path=/.meguca_build/src/github.com/bakape
	export GOPATH=/.meguca_build
	export PKG_CONFIG_PATH:=$(PKG_CONFIG_PATH):/mingw64/lib/pkgconfig/
	export PKG_CONFIG_LIBDIR=/mingw64/lib/pkgconfig/
	export PATH:=$(PATH):/mingw64/bin/
	binary=meguca.exe
	is_windows=true
else
	build_path=$(PWD)/.build/src/github.com/bakape
	export GOPATH=$(PWD)/.build
	binary=meguca
	is_windows=false
endif

.PHONY: server client imager

all: server client

client: client_vendor
	$(gulp)

client_deps:
	npm install --progress false --depth 0

watch:
	$(gulp) -w

client_vendor: client_deps
	mkdir -p www/js/vendor
	cp node_modules/dom4/build/dom4.js node_modules/core-js/client/core.min.js node_modules/core-js/client/core.min.js.map node_modules/babel-polyfill/dist/polyfill.min.js node_modules/proxy-polyfill/proxy.min.js www/js/vendor
	 $(uglifyjs) node_modules/whatwg-fetch/fetch.js -o www/js/vendor/fetch.js
	 $(uglifyjs) node_modules/almond/almond.js -o www/js/vendor/almond.js

server: server_deps generate imager
	go build -v -o $(binary)
ifeq ($(is_windows), true)
	cp /mingw64/bin/*.dll ./
endif

imager:
	$(MAKE) -C imager/lib

generate:
	$(MAKE) -C templates

server_deps: build_dirs
	go get -v github.com/valyala/quicktemplate/qtc
	go list -f '{{.Deps}}' . \
		| tr "[" " " \
		| tr "]" " " \
		| xargs go list -e -f '{{if not .Standard}}{{.ImportPath}}{{end}}' \
		| grep -v 'github.com/bakape/meguca' \
		| xargs go get -v

update_deps: build_dirs
	go get -u -v github.com/valyala/quicktemplate/qtc
	go list -f '{{.Deps}}' . \
		| tr "[" " " \
		| tr "]" " " \
		| xargs go list -e -f '{{if not .Standard}}{{.ImportPath}}{{end}}' \
		| grep -v 'github.com/bakape/meguca' \
		| xargs go get -v -u
	npm update

build_dirs:
ifeq ($(is_windows), true)
	rm -rf $(build_path)
endif
	mkdir -p $(build_path)
	ln -sfn "$(shell pwd)" $(build_path)/meguca

client_clean:
	rm -rf www/js www/css/*.css www/css/maps www/lang node_modules

clean: client_clean
	rm -rf .build .ffmpeg .package meguca-*.zip meguca-*.tar.xz meguca meguca.exe
	$(MAKE) -C imager/lib clean
	$(MAKE) -C templates clean
ifeq ($(is_windows), true)
	rm -rf /.meguca_build *.dll
endif

dist_clean: clean
	rm -rf images error.log

test: server_deps
	go test -p 1 ./...
