export node_bins=$(PWD)/node_modules/.bin
export uglifyjs=$(node_bins)/uglifyjs
export gulp=$(node_bins)/gulp
export is_windows=false
binary=meguca

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
	$(MAKE) -C client_cpp
	rm -f www/wasm/main.*
	cp client_cpp/*.wasm client_cpp/*.js www/wasm
ifeq ($(DEBUG),1)
	cp client_cpp/*.wast client_cpp/*.wasm.map www/wasm
endif

wasm_clean:
	$(MAKE) -C client_cpp clean
	rm -f www/wasm/*.js www/wasm/*.wasm www/wasm/*.map www/wasm/*.wast

watch:
	$(gulp) -w

client_vendor: client_deps
	mkdir -p www/js/vendor
	cp node_modules/dom4/build/dom4.js node_modules/core-js/client/core.min.js node_modules/core-js/client/core.min.js.map www/js/vendor
	$(uglifyjs) node_modules/almond/almond.js -o www/js/vendor/almond.js

css:
	$(gulp) css

server:
	cd server; go get -v
	cd server; go build -v -o ../$(binary)
ifeq ($(is_windows), true)
	cp /mingw64/bin/*.dll ./
endif

server_no_fetch:
	cd server; go build -v -o ../$(binary)

generate: generate_clean
	go get -v github.com/valyala/quicktemplate/qtc github.com/jteeuwen/go-bindata/... github.com/mailru/easyjson/... github.com/bakape/thumbnailer github.com/gorilla/websocket
	go generate ./server/...

generate_clean:
	rm -f server/db/bin_data.go server/lang/bin_data.go server/assets/bin_data.go
	rm -f server/common/*_easyjson.go
	rm -f server/config/*_easyjson.go
	rm -f server/websockets/feeds/*_easyjson.go
	rm -f server/templates/*.qtpl.go

client_clean:
	rm -rf www/js www/css/*.css www/css/maps node_modules

clean: client_clean wasm_clean generate_clean
	rm -rf .build .ffmpeg .package target meguca-*.zip meguca-*.tar.xz meguca meguca.exe
ifeq ($(is_windows), true)
	rm -rf /.meguca_build *.dll
endif

dist_clean: clean
	rm -rf images error.log db.db

test:
	cd server; go test --race -p 1 ./...

test_no_race:
	cd server; go test -p 1 ./...

check: test

