.PHONY: server client imager test websockets

all: server client css websockets

client:
	$(MAKE) -C lang all
	$(MAKE) -C client all

install_tools:
	go get -u github.com/valyala/quicktemplate \
		github.com/rakyll/statik \
		github.com/valyala/quicktemplate/qtc

css:
ifneq ($(NO_DEPS),1)
	npm install --progress false --depth 0
endif
	$(MAKE) -C less

generate:
	go generate ./...

websockets:
# Generate a hash and add it to LDFLAGS of the binary to force a rebuild on the
# Go side
	rm -f websockets/libwebsockets*.a
	cargo build $(if $(filter 1,$(DEBUG)),, --release)
		SRC=target/$(if $(filter 1,$(DEBUG)),debug,release)/libwebsockets.a; \
		HASH=$$(md5sum $$SRC | cut -c 1-4); \
		cp $$SRC websockets/libwebsockets_$$HASH.a  && \
		/bin/echo -e "package websockets\n\n// #cgo LDFLAGS: -L\$${SRCDIR} -lwebsockets_$$HASH\nimport \"C\"" > ./websockets/lib_hash.go

server: websockets
	go build -v

clean:
	rm -rf meguca websockets/libwebsockets*.a
	cargo clean
	rm -rf www/css/*.css www/css/*.css.map node_modules
	$(MAKE) -C client clean

# TODO: Minify language pack JSON

test: websockets
	cargo test
	$(MAKE) -C client test
	go test --race ./...

test_no_race:
	go test ./...

test_docker:
	docker-compose build
	docker-compose run --rm -e CI=true meguca make test

