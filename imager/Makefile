all: findapng

PNGFLAGS := $(shell pkg-config libpng --cflags)
PNGLIBS := $(shell pkg-config libpng --libs)

findapng: findapng.c
	gcc -O2 $(PNGFLAGS) -o $@ $^ $(PNGLIBS)


.PHONY: all clean

clean:
	rm -rf -- findapng
