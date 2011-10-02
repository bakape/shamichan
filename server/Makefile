CLIENT_SRC := $(shell node server.js --show-config CLIENT_DEPS)
CLIENT_JS := ../www/js/client-$(shell node server.js --client-version).js
DEBUG := $(shell node server.js --show-config DEBUG)

all: tripcode.node client

jsmin: jsmin.c
	gcc -o $@ $^

tripcode.node: .build tripcode.cc
	node-waf build
	@cp .build/default/$@ $@

.build: wscript
	node-waf configure

ifeq ($(DEBUG),true)
CLIENT_JS := $(CLIENT_JS:%.js=%.debug.js)
endif
$(CLIENT_JS): $(CLIENT_SRC) jsmin
	node make_client.js $(CLIENT_SRC) > $@

client: $(CLIENT_JS)

.PHONY: all client clean

clean:
	rm -rf -- .build jsmin tripcode.node ../www/js/client-*.js
