
all: tripcode.node

tripcode.node: build tripcode.cc
	node-gyp build
	@cp build/Release/$@ $@

build: binding.gyp
	node-gyp configure

# Legacy build method: `make waf`
waf: .build
	node-waf build
	@cp .build/*/tripcode.node tripcode.node

.build: wscript
	node-waf configure

.PHONY: all clean waf

clean:
	rm -rf -- .build build tripcode.node
