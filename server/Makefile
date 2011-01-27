CLIENT_SRC = lib/socket.io.js common.js client.js
RELEASE_SRC = lib/mixpanel.js
LAST_SRC = lib/mixpanel_invoke.js guns.js
CLIENT_JS := ../www/js/client-$(shell node config.js --version).js
DEBUG := $(shell node config.js --show-config DEBUG)

all: builder tripcode.node client

builder: builder.c
	gcc -o $@ $^

jsmin: jsmin.c
	gcc -o $@ $^

tripcode.node: .build tripcode.cc
	node-waf build
	@cp .build/default/$@ $@

.build: wscript
	node-waf configure

ifeq ($(DEBUG),true)
$(CLIENT_JS): $(CLIENT_SRC) $(LAST_SRC) jsmin
	@node make_client.js $(CLIENT_SRC) $(LAST_SRC) > $@
else
$(CLIENT_JS): $(CLIENT_SRC) $(RELEASE_SRC) $(LAST_SRC) jsmin
	@node make_client.js $(CLIENT_SRC) $(RELEASE_SRC) $(LAST_SRC) \
		> $@.orig && ./jsmin < $@.orig > $@
	@rm -f $@.orig
endif

client: $(CLIENT_JS)

.PHONY: all client clean

clean:
	rm -rf -- .build builder jsmin tripcode.node ../www/js/client-*.js
