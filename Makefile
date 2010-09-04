all: www/js/client.js

www/js/client.js: client.js common.js config.js
	@echo Building $@.
	@cp config.js $@
	@echo >> $@
	@sed "s/^exports\.//g" common.js >> $@
	@echo >> $@
	@cat client.js >> $@
