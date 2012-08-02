
all: tripcode.node

tripcode.node: .build tripcode.cc
	node-waf build
	@cp .build/*/$@ $@

.build: wscript
	node-waf configure

.PHONY: all clean

clean:
	rm -rf -- .build tripcode.node
