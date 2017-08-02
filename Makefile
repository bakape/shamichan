export node_bins=$(PWD)/node_modules/.bin
export uglifyjs=$(node_bins)/uglifyjs
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
	cargo build --target=wasm32-unknown-emscripten --release
	cp `ls -S target/wasm32-unknown-emscripten/release/deps/client*.wasm | tail -n 1` www/wasm/main.wasm
	cp `ls -S target/wasm32-unknown-emscripten/release/deps/client*.asm.js | tail -n 1` www/wasm/main.asm.js
	sed 's/client-[0-9a-f]\{16\}\./main\./g' target/wasm32-unknown-emscripten/release/client.js > www/wasm/main.js

wasm_debug:
	mkdir -p www/wasm
	cargo build --target=wasm32-unknown-emscripten
	cp `ls -S target/wasm32-unknown-emscripten/debug/deps/client*.wasm | tail -n 1` www/wasm/main.wasm
	cp `ls -S target/wasm32-unknown-emscripten/debug/deps/client*.asm.js | tail -n 1` www/wasm/main.asm.js
	sed 's/client-[0-9a-f]\{16\}\./main\./g' target/wasm32-unknown-emscripten/debug/client.js > www/wasm/main.js

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
