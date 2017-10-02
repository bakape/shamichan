all: $(addsuffix .bc, $(basename $(wildcard *.cc)))

%.bc: %.cc
	emcc $(EMCCFLAGS) $^ -o $@

clean:
	rm -f *.bc
