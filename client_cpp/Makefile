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
COMPILE_FLAGS:=$(COMPILE_FLAGS) -I$(abspath ./json/include)
export EMCCFLAGS=$(COMPILE_FLAGS) -Wall -Wextra -Wno-switch -Wno-unused-parameter

all: clean_output
	$(MAKE) -C brunhild
	$(MAKE) -C src
	emcc src/*.bc src/page/*.bc src/posts/*.bc src/options/*.bc src/connection/*.bc brunhild/*.bc -o linked.bc $(COMPILE_FLAGS) $(LTO) $(SETTINGS)
ifeq ($(DEBUG),0)
	emcc linked.bc -o main.js --separate-asm -Wno-separate-asm $(COMPILE_FLAGS) $(LTO) $(SETTINGS)
endif
	emcc linked.bc -o main.js -s WASM=1 $(COMPILE_FLAGS) $(LTO) $(SETTINGS)

clean_output:
	rm -f *.wasm *.wast *.js *.wasm.map *.js

clean: clean_output
	rm -f *.bc
	$(MAKE) -C brunhild clean
	$(MAKE) -C src clean
