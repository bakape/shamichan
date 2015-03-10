all:
	$(MAKE) -C imager
	$(MAKE) -C tripcode

.PHONY: all clean

clean:
	rm -rf --  state www/js/client-*.js www/js/vendor-*.js www/css/*.css
	$(MAKE) -C imager -w clean
	$(MAKE) -C tripcode -w clean
