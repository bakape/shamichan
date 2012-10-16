all: findapng perceptual

PNGFLAGS := $(shell pkg-config libpng --cflags)
PNGLIBS := $(shell pkg-config libpng --libs)

findapng: findapng.c
	gcc -O2 $(PNGFLAGS) -o $@ $^ $(PNGLIBS)

perceptual: perceptual.c
	gcc -O2 -o $@ $^

.PHONY: all clean

clean:
	rm -rf -- findapng perceptual
