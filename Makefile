export node_bins=$(PWD)/node_modules/.bin
export uglifyjs=$(node_bins)/uglifyjs --mangle safari10=true
export gulp=$(node_bins)/gulp
export is_windows=false
binary=meguca
ifeq ($(GOPATH),)
	export PATH:=$(PATH):$(HOME)/go/bin
	export GOPATH=$(HOME)/go:$(PWD)/go
else
	export PATH:=$(PATH):$(GOPATH)/bin
	export GOPATH:=$(GOPATH):$(PWD)/go
endif

# Differentiate between Unix and mingw builds
ifeq ($(OS), Windows_NT)
	export PKG_CONFIG_PATH:=$(PKG_CONFIG_PATH):/mingw64/lib/pkgconfig/
	export PKG_CONFIG_LIBDIR=/mingw64/lib/pkgconfig/
	export PATH:=$(PATH):/mingw64/bin/
	export is_windows=true
	binary=meguca.exe
endif

.PHONY: server client imager test

all: server client

client: client_vendor
	$(gulp)

client_deps:
	npm install --progress false --depth 0

wasm:
	mkdir -p www/wasm
	emcc \
		-std=c++1z \
		--bind \
		-Oz --closure 1 -s MODULARIZE=1 \
		-s NO_EXIT_RUNTIME=1 \
		-s WASM=1 \
		client_cpp/main.cpp -o www/wasm/main.js

# TODO: Generate asm.js fallback. This can be done by compiling to .bc and then
# conpiling that to .wasm and .ams.js
# emcc --separate-asm -Wno-separate-asm -Oz --closure 2 -s MODULARIZE=1 -s WASM=1 -s ASM_JS=1 client_cpp/main.cpp -o www/wasm/main.js

wasm_debug:
	mkdir -p www/wasm
	emcc \
		-std=c++1z \
		--bind \
		-g \
		-s NO_EXIT_RUNTIME=1 \
		-s WASM=1 \
		client_cpp/main.cpp -o www/wasm/main.js

watch:
	$(gulp) -w

client_vendor: client_deps
	mkdir -p www/js/vendor
	cp node_modules/dom4/build/dom4.js node_modules/core-js/client/core.min.js node_modules/core-js/client/core.min.js.map node_modules/babel-polyfill/dist/polyfill.min.js node_modules/proxy-polyfill/proxy.min.js www/js/vendor
	$(uglifyjs) node_modules/whatwg-fetch/fetch.js -o www/js/vendor/fetch.js
	$(uglifyjs) node_modules/almond/almond.js -o www/js/vendor/almond.js

server: generate server_deps
	go build -v -o $(binary) meguca
ifeq ($(is_windows), true)
	cp /mingw64/bin/*.dll ./
endif

generate:
	go get -v github.com/valyala/quicktemplate/qtc github.com/jteeuwen/go-bindata/... github.com/mailru/easyjson/...
	rm -f go/src/meguca/common/*_easyjson.go
	rm -f go/src/meguca/config/*_easyjson.go
	rm -f go/src/meguca/templates/*.qtpl.go
	go generate meguca/...

server_deps:
	go list -f '{{.Deps}}' meguca | tr -d '[]' | xargs go get -v

update_deps:
	go get -u -v github.com/valyala/quicktemplate/qtc github.com/jteeuwen/go-bindata/... github.com/mailru/easyjson/...
	go list -f '{{.Deps}}' meguca | tr -d '[]' | xargs go list -e -f '{{if not .Standard}}{{.ImportPath}}{{end}}' | grep -v 'meguca' | xargs go get -u -v
	npm update

client_clean:
	rm -rf www/js www/wasm www/css/*.css www/css/maps www/lang node_modules

clean: client_clean
	rm -rf .build .ffmpeg .package target meguca-*.zip meguca-*.tar.xz meguca meguca.exe
	$(MAKE) -C scripts/migration/3to4 clean
ifeq ($(is_windows), true)
	rm -rf /.meguca_build *.dll
endif

dist_clean: clean
	rm -rf images error.log

test:
	go test --race -p 1 meguca/...

test_no_race:
	go test -p 1 meguca/...

upgrade_v4: generate
	go get -v github.com/dancannon/gorethink
	$(MAKE) -C scripts/migration/3to4 upgrade
