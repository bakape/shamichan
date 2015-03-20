
all: tripcode.node

tripcode.node: build tripcode.cc
	node-gyp build
	@cp build/Release/$@ $@

build: binding.gyp
	node-gyp configure

.PHONY: all clean

clean:
	rm -rf -- .build build tripcode.node
