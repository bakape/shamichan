all: findapng compare.node

PNGFLAGS := $(shell pkg-config libpng --cflags)
PNGLIBS := $(shell pkg-config libpng --libs)

findapng: findapng.c
	gcc -O2 $(PNGFLAGS) -o $@ $^ $(PNGLIBS)

compare.node: build compare.cpp
	node-gyp build
	@cp build/Release/$@ $@

build: binding.gyp
	node-gyp configure

.PHONY: all clean

clean:
	rm -rf -- findapng .build build compare.node
