CLIENT_SRC := $(shell node get.js CLIENT_DEPS)
DEBUG := $(shell node get.js DEBUG)
ifeq ($(DEBUG),true)
CLIENT_JS := www/js/client.debug.js
else
CLIENT_JS := www/js/client-$(shell node get.js --client-version).js
endif

all: client
	$(MAKE) -C imager
	$(MAKE) -C tripcode

jsmin: lib/jsmin.c
	gcc -o $@ $^

$(CLIENT_JS): $(CLIENT_SRC) jsmin config.js deps.js
	node make_client.js $(CLIENT_SRC) > $@

client: $(CLIENT_JS)

modjs:
	node make_client.js $(shell node get.js MOD_CLIENT_DEPS)

.PHONY: all client clean modjs

clean:
	rm -rf -- .build jsmin www/js/client{.,-}*.js
	$(MAKE) -C imager -w clean
	$(MAKE) -C tripcode -w clean
