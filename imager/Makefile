all: findapng compare

PNGFLAGS := $(shell pkg-config libpng --cflags)
PNGLIBS := $(shell pkg-config libpng --libs)

findapng: findapng.c
	gcc -O2 $(PNGFLAGS) -o $@ $^ $(PNGLIBS)

compare: compare.c
	gcc -O2 -o $@ $^

.PHONY: all clean

clean:
	rm -rf -- findapng compare
