export EMCCFLAGS=-std=c++11 -s NO_EXIT_RUNTIME=1
LTO=
ifeq ($(DEBUG),1)
	export EMCCFLAGS+= -g4 -s DEMANGLE_SUPPORT=1
else
	export EMCCFLAGS+= -Oz -g0
	LTO=--llvm-lto 3
endif

all: clean_output main.bc
	$(MAKE) -C brunhild
	emcc main.bc brunhild/*.bc -o linked.bc $(EMCCFLAGS) $(LTO)
ifneq ($(DEBUG),1)
	emcc linked.bc -o main.js --separate-asm -Wno-separate-asm $(EMCCFLAGS)
endif
	emcc linked.bc -o main.js -s WASM=1 $(EMCCFLAGS)

main.bc:
	emcc $(EMCCFLAGS) main.cc -o main.bc

clean_output:
	rm -f *.wasm *.wast *.js *.wasm.map *.js

clean: clean_output
	rm -rf *.bc
	$(MAKE) -C brunhild clean
