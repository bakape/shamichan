CLIENT_JS = ../www/js/client-v$(shell node config.js --show-config VERSION).js
RELEASE_JS = $(CLIENT_JS:%.js=%.min.js)

all: builder tripcode.node

release: $(RELEASE_JS)

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
	@sed "s/^exports\.//g" common.js > $@
	@echo >> $@
	@cat client.js >> $@

$(RELEASE_JS): $(CLIENT_JS)
	./jsmin < $< > $@

clean:
	rm -rf -- .build builder jsmin tripcode.node ../www/js/client-v*.js
