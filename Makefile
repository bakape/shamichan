all: builder

builder: builder.c
	gcc -o $@ $^

www/js/client.js: client.js common.js config.js
	@cp config.js $@
	@echo >> $@
	@sed "s/^exports\.//g" common.js >> $@
	@echo >> $@
	@cat client.js >> $@
