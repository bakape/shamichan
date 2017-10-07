# -Winvalid-pp-token triggers on empty JS strings in macros
WARNINGS=-Wall -Wextra -Wno-sign-compare -Wno-invalid-pp-token
export EMCCFLAGS=-std=c++1z --bind  -s NO_EXIT_RUNTIME=1 $(WARNINGS)
LTO=
ifeq ($(DEBUG),1)
	# -O0 is too large to load in the browser. Fails with:
	# LinkError: table import 128 has maximum larger than maximum 24960
	export EMCCFLAGS+= -O1 -g4 -s DEMANGLE_SUPPORT=1
else
	export EMCCFLAGS+= -Oz -g0
	LTO=--llvm-lto 3
endif

all: clean_output
	$(MAKE) -C brunhild
	$(MAKE) -C src
	emcc src/*.bc brunhild/*.bc -o linked.bc $(EMCCFLAGS) $(LTO)
ifneq ($(DEBUG),1)
	emcc linked.bc -o main.js --separate-asm -Wno-separate-asm $(EMCCFLAGS)
endif
	emcc linked.bc -o main.js -s WASM=1 $(EMCCFLAGS)

clean_output:
	rm -f *.wasm *.wast *.js *.wasm.map *.js

clean: clean_output
	$(MAKE) -C brunhild clean
	$(MAKE) -C src clean
