
all: perceptual tripcode.node

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
