.PHONY: client server

all: client server

install_tools:
	$(MAKE) -C client install_tools

client:
	$(MAKE) -C client all

client_watch:
	$(MAKE) -C client watch

server:
	$(MAKE) -C server all

debug:
	$(MAKE) -C client all
	$(MAKE) -C server debug
