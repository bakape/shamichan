CLIENT_JS := ../www/js/client-v$(shell node config.js --show-config VERSION).js
DEBUG := $(shell node config.js --show-config DEBUG)

all: builder tripcode.node

builder: builder.c
	gcc -o $@ $^

jsmin: jsmin.c
	gcc -o $@ $^

tripcode.node: .build tripcode.cc
	node-waf build
	@cp .build/default/$@ $@

.build: wscript
	node-waf configure

$(CLIENT_JS): client.js common.js jsmin
ifeq ($(DEBUG),true)
	@node make_client.js common.js client.js > $@
else
	@node make_client.js common.js client.js > $@.orig && \
		./jsmin < $@.orig > $@
	@rm -f $@.orig
endif

clean:
	rm -rf -- .build builder jsmin tripcode.node ../www/js/client-v*.js
