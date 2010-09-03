all: www/js/client.js

www/js/client.js: client.js common.js
	@echo Building $@.
	@sed "s/^exports\.//g" common.js > $@
	@echo >> $@
	@cat client.js >> $@
