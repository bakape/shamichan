all: $(addsuffix .bc, $(basename $(wildcard *.cpp)))

%.bc: %.cpp
	emcc $(EMCCFLAGS) $^ -o $@.bc

clean:
	rm -f *.bc
