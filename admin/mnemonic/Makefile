all: mnemonics.node

mnemonics.node: build mnemonics.cpp
	node-gyp build
	@cp build/Release/$@ $@

build: binding.gyp
	node-gyp configure

.PHONY: all clean

clean:
	rm -rf -- .build build mnemonics.node
