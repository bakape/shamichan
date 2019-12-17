.PHONY: server client imager test websockets

all: server client css

client:
	$(MAKE) -C client all

install_tools:
	$(MAKE) -C client install_tools

css:
ifneq ($(NO_DEPS),1)
	npm install --progress false --depth 0
endif
	$(MAKE) -C less

generate:
	go generate ./...

websockets:
	cargo build $(if $(DEBUG),, --release)
	cp target/$(if $(DEBUG),debug,release)/libwebsockets.a websockets/

server: generate websockets
	go build -v

clean:
	rm -rf meguca
	cargo clean
	rm -rf www/css/*.css www/css/*.css.map node_modules
	$(MAKE) -C client clean

# TODO: Compress language pack JSON

test: websockets
	cargo test
	go test --race ./...

test_no_race:
	go test ./...

test_docker:
	docker-compose build
	docker-compose ru` --rm -e CI=true meguca make test

