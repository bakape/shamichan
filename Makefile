.PHONY: server client imager test websockets

all: server client css websockets

client:
ifneq ($(NO_DEPS),1)
	npm install --progress false --depth 0
endif
	$(MAKE) -C lang all
	$(MAKE) -C client all

client_watch:
	DEBUG=1 $(MAKE) css client
	while inotifywait \
		-e modify,delete \
		-q \
		-r \
		client/src protocol/src less; \
	do \
		DEBUG=1 NO_DEPS=1 $(MAKE) css client; \
	done

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
	cargo build \
		--workspace \
		--exclude client\
		$(if $(filter 1,$(DEBUG)),, --release)
	SRC=target/$(if $(filter 1,$(DEBUG)),debug,release)/libwebsockets.a; \
		HASH=$$(md5sum $$SRC | cut -c 1-4); \
		cp $$SRC websockets/libwebsockets_$$HASH.a  && \
		/bin/echo -e "package websockets\n\n// #cgo LDFLAGS: -L\$${SRCDIR} -lwebsockets_$$HASH\nimport \"C\"" > ./websockets/lib_hash.go

server: websockets
	go build -v

clean:
	rm -rf meguca websockets/libwebsockets*.a www/client www/js
	cargo clean
	rm -rf www/css/*.css www/css/*.css.map node_modules
	$(MAKE) -C client clean

test: websockets
	cargo test
	go test --race ./...

test_no_race: websockets
	cargo test
	go test ./...

release: test
	docker build -t meguca .
	docker tag meguca bakape/meguca:`git describe --tags`
	docker tag meguca bakape/meguca:latest
	docker push bakape/meguca
