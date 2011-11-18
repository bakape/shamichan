CLIENT_SRC := $(shell node server.js --show-config CLIENT_DEPS)
DEBUG := $(shell node server.js --show-config DEBUG)
ifeq ($(DEBUG),true)
CLIENT_JS := ../www/js/client.debug.js
else
CLIENT_JS := ../www/js/client-$(shell node server.js --client-version).js
endif

all: perceptual tripcode.node client

jsmin: jsmin.c
	gcc -o $@ $^

perceptual: perceptual.c
	gcc -O2 -o $@ $^

tripcode.node: .build tripcode.cc
	node-waf build
	@cp .build/default/$@ $@

test:
	node ghost.js

.build: wscript
	node-waf configure

$(CLIENT_JS): $(CLIENT_SRC) jsmin
	node make_client.js $(CLIENT_SRC) > $@

client: $(CLIENT_JS)

.PHONY: all client clean

clean:
	rm -rf -- .build jsmin tripcode.node ../www/js/client{.,-}*.js
