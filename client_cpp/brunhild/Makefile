all: $(addsuffix .bc, $(basename $(wildcard *.cpp)))

%.bc: %.cpp
	emcc $(EMCCFLAGS) $^ -o $@

clean:
	rm -f *.bc
