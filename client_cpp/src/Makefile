all: page posts options $(addsuffix .bc, $(basename $(wildcard *.cc)))

.PHONY: page posts options

page:
	$(MAKE) -C page

posts:
	$(MAKE) -C posts

options:
	$(MAKE) -C options

%.bc: %.cc
	emcc $^ -o $@ $(EMCCFLAGS)

clean:
	rm -f *.bc
	$(MAKE) -C page clean
	$(MAKE) -C posts clean
	$(MAKE) -C options clean
