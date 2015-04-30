all: bootstrap client
	$(MAKE) -C imager
	$(MAKE) -C server/tripcode

client: FORCE
	./node_modules/gulp/bin/gulp.js -- client mod css

FORCE:

.PHONY: all clean

bootstrap:
	./scripts/bootstrap.sh

upgrade: clean
	rm -rf -- ./node_modules
	npm -- install

clean:
	$(MAKE) -C imager -w clean
	$(MAKE) -C server/tripcode -w clean

client_clean:
	rm -rf -- state www/js/client*.js* www/js/vendor-*.js www/js/alpha*.js* www/css/*.css
