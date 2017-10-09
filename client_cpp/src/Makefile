all: page $(addsuffix .bc, $(basename $(wildcard *.cc)))

.PHONY: page

page:
	$(MAKE) -C page

%.bc: %.cc
	emcc $(EMCCFLAGS) $^ -o $@

clean:
	rm -f *.bc
	$(MAKE) -C page clean
