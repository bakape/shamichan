all: page posts $(addsuffix .bc, $(basename $(wildcard *.cc)))

.PHONY: page posts

page:
	$(MAKE) -C page

posts:
	$(MAKE) -C posts

%.bc: %.cc
	emcc $(EMCCFLAGS) $^ -o $@

clean:
	rm -f *.bc
	$(MAKE) -C page clean
	$(MAKE) -C posts clean
