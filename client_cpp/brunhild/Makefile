all: $(addsuffix .bc, $(basename $(wildcard *.cc)))

%.bc: %.cc
	emcc $^ -o $@ $(EMCCFLAGS)

clean:
	rm -f *.bc
