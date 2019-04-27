export node_bins=$(PWD)/node_modules/.bin
export uglifyjs=$(node_bins)/uglifyjs
export gulp=$(node_bins)/gulp
export is_windows=false
export GO111MODULE=on

# Differentiate between Unix-like and mingw builds
ifeq ($(OS), Windows_NT)
	export PKG_CONFIG_PATH:=$(PKG_CONFIG_PATH):/mingw64/lib/pkgconfig/
	export PKG_CONFIG_LIBDIR=/mingw64/lib/pkgconfig/
	export PATH:=$(PATH):/mingw64/bin/
	export is_windows=true
endif

.PHONY: server client imager test

all: server client

client: client_vendor
	$(gulp)

client_deps:
	npm install --progress false --depth 0

watch:
	$(gulp) -w

client_vendor: client_deps
	mkdir -p www/js/vendor
	cp node_modules/dom4/build/dom4.js node_modules/core-js/client/core.min.js node_modules/core-js/client/core.min.js.map www/js/vendor
	$(uglifyjs) node_modules/almond/almond.js -o www/js/vendor/almond.js

css:
	$(gulp) css

generate:
	go generate ./...

server:
	go build -v

client_clean:
	rm -rf www/js www/css/*.css www/css/maps node_modules

clean: client_clean
	rm -rf .build .ffmpeg .package target meguca-*.zip meguca-*.tar.xz meguca meguca.exe server/pkg
ifeq ($(is_windows), true)
	rm -rf /.meguca_build *.dll
endif

test:
	go test --race ./...

test_no_race:
	go test ./...

test_docker:
	docker build -t meguca_test .
	docker run -t --rm --entrypoint scripts/docker_test.sh -e CI meguca_test

