all: findapng perceptual

findapng: findapng.c
	gcc -O2 -o $@ $(shell pkg-config libpng --cflags --libs) $^

perceptual: perceptual.c
	gcc -O2 -o $@ $^

.PHONY: all clean

clean:
	rm -rf -- findapng perceptual
