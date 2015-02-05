all: findapng.node compare.node

findapng.node: build findapng.cpp
	node-gyp build
	@cp build/Release/$@ $@

compare.node: build compare.cpp
	node-gyp build
	@cp build/Release/$@ $@

build: binding.gyp
	node-gyp configure

.PHONY: all clean

clean:
	rm -rf -- .build build compare.node findapng.node
