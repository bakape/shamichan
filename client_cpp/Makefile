SETTINGS=-s NO_EXIT_RUNTIME=1 -s TOTAL_MEMORY=67108864 -s ALLOW_MEMORY_GROWTH=1 -Wno-almost-asm -s NO_FILESYSTEM=1
COMPILE_FLAGS=-std=c++1z --bind

ifeq ($(DEBUG),)
	DEBUG=0
endif

ifeq ($(DEBUG),0)
	COMPILE_FLAGS+=-O3 -g0
	LTO=--llvm-lto 3
else
	COMPILE_FLAGS+=-g4
	SETTINGS+=-s DEMANGLE_SUPPORT=1 -s ASSERTIONS=2 -s DISABLE_EXCEPTION_CATCHING=0
	ifeq ($(DEBUG),1)
		COMPILE_FLAGS+=-O1
	endif
endif
export EMCCFLAGS=$(COMPILE_FLAGS) -Wall -Wextra -Wno-switch -Wno-unused-parameter -Werror

all: clean_output
	$(MAKE) -C src
	emcc -o linked.bc $(COMPILE_FLAGS) $(LTO) $(SETTINGS)
	emcc linked.bc -o wasm.js -s WASM=1 $(COMPILE_FLAGS) $(LTO) $(SETTINGS)

clean_output:
	rm -f *.wasm *.wast *.js *.wasm.map *.js

clean: clean_output
	rm -f *.bc
	$(MAKE) -C src clean
