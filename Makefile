.PHONY: server client imager test websockets

# TODO: build imager

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
		client/src common/src less; \
	do \
		DEBUG=1 NO_DEPS=1 $(MAKE) css client; \
	done

css:
ifneq ($(NO_DEPS),1)
	npm install --progress false --depth 0
endif
	$(MAKE) -C less

server:
	SQLX_OFFLINE=true cargo build \
		--workspace \
		--exclude client \
		$(if $(filter 1,$(DEBUG)),,--release)
	cp target/$(if $(filter 1,$(DEBUG)),debug,release)/meguca meguca

clean:
	rm -rf meguca www/client www/js
	cargo clean
	rm -rf www/css/*.css www/css/*.css.map node_modules
	$(MAKE) -C client clean

test:
	cargo test
	# go test --race ./...

# Prepare offline version of checked queries for compilation without a connected
# database
db_prepare_offline:
	cargo sqlx prepare -- --package server
	mv sqlx-data.json server/

test_no_race:
	cargo test
	# go test ./...
