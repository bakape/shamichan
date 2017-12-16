all: page posts options connection $(addsuffix .bc, $(basename $(wildcard *.cc)))

.PHONY: page posts options connection

page:
	$(MAKE) -C page

posts:
	$(MAKE) -C posts

options:
	$(MAKE) -C options

connection:
	$(MAKE) -C connection

%.bc: %.cc
	emcc $^ -o $@ $(EMCCFLAGS)

clean:
	rm -f *.bc
	$(MAKE) -C page clean
	$(MAKE) -C posts clean
	$(MAKE) -C options clean
