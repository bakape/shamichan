
all: findapng perceptual tripcode.node

findapng: findapng.c
	gcc -O2 -o $@ $(shell pkg-config libpng --cflags --libs) $^

perceptual: perceptual.c
	gcc -O2 -o $@ $^

tripcode.node: .build tripcode.cc
	node-waf build
	@cp .build/*/$@ $@

.build: wscript
	node-waf configure

.PHONY: all clean

clean:
	rm -rf -- .build perceptual tripcode.node
