all: bootstrap client
	$(MAKE) -C imager
	$(MAKE) -C tripcode
	

client: FORCE
	./node_modules/gulp/bin/gulp.js -- client mod vendor css alpha

FORCE:

.PHONY: all clean

bootstrap:
	./upkeep/bootstrap.sh

upgrade: clean
	rm -rf -- ./node_modules
	npm -- install

clean:
	$(MAKE) -C imager -w clean
	$(MAKE) -C tripcode -w clean

client_clean:
	rm -rf --  state www/js/client-*.js www/js/vendor-*.js www/js/alpha-*.js* www/css/*.css
