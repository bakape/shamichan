all: bootstrap client
	$(MAKE) -C imager
	$(MAKE) -C server/tripcode
	$(MAKE) -C admin/mnemonic

client: FORCE
	./node_modules/gulp/bin/gulp.js -- client vendor mod css lang legacy

FORCE:

.PHONY: all clean

bootstrap:
	node ./scripts/bootstrap

upgrade: clean
	rm -rf -- ./node_modules
	npm -- install

clean:
	$(MAKE) -C imager -w clean
	$(MAKE) -C server/tripcode -w clean
	$(MAKE) -C admin/mnemonic -w clean

client_clean:
	rm -rf -- state www/js/{client,vendor,legacy}.js* www/css/{*.css,maps}\
		www/js/lang
