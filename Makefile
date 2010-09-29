CLIENT_JS = www/js/client-v$(shell node config.js --show-config VERSION).js

all: builder tripcode.node

builder: builder.c
	gcc -o $@ $^

tripcode.node: .build tripcode.cc
	node-waf build
	@cp .build/default/$@ $@

.build: wscript
	node-waf configure

$(CLIENT_JS): client.js common.js
	@sed "s/^exports\.//g" common.js > $@
	@echo >> $@
	@cat client.js >> $@

clean:
	rm -rf -- .build builder tripcode.node www/js/client-v*.js
