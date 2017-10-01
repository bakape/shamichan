export EMCCFLAGS=-std=c++1z -s NO_EXIT_RUNTIME=1
LTO=
ifeq ($(DEBUG),1)
	export EMCCFLAGS+= -g4
else
	export EMCCFLAGS+= -Oz --closure 1 -s MODULARIZE=1 -g0
	LTO=--llvm-lto 3
endif

all: clean main.bc
	$(MAKE) -C brunhild
	emcc main.bc brunhild/*.bc -o linked.bc $(EMCCFLAGS) $(LTO)
ifneq ($(DEBUG),1)
	emcc linked.bc -o main.js --separate-asm -Wno-separate-asm $(EMCCFLAGS)
endif
	emcc linked.bc -o main.js -s WASM=1 $(EMCCFLAGS)

main.bc:
	emcc $(EMCCFLAGS) main.cpp -o main.bc

clean:
	rm -f *.wasm *.wast *.js *.wasm.map *.js *.bc
	$(MAKE) -C brunhild clean
