all:
	$(MAKE) -C imager
	$(MAKE) -C tripcode

client:
	@echo 'make client' is no longer necessary
	@false

.PHONY: all clean client

clean:
	rm -rf -- .build state www/js/client{.,-}*.js
	$(MAKE) -C imager -w clean
	$(MAKE) -C tripcode -w clean
