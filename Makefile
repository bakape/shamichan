.PHONY: server client imager test websockets

all: server client

client:
	$(MAKE) -C client all

install_tools:
	$(MAKE) -C client install_tools

# TODO: Build without gulp
# css:
# 	$(gulp) css

generate:
	go generate ./...

websockets:
	cargo build --release
	cp target/release/libwebsockets.a websockets/

server: generate websockets
	go build -v

clean:
	rm -rf meguca
	cargo clean
	rm -rf www/js www/css/*.css www/css/maps node_modules
	$(MAKE) -C client clean

test: websockets
	cargo test
	go test --race ./...

test_no_race:
	go test ./...

test_docker:
	docker-compose build
	docker-compose ru` --rm -e CI=true meguca make test

