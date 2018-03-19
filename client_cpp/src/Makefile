DIRS=$(subst /,,$(wildcard */))
.PHONY: all $(DIRS)

all: $(DIRS) $(addsuffix .bc, $(basename $(wildcard *.cc)))

$(DIRS):
	$(MAKE) -C $@

%.bc: %.cc
	emcc $^ -o $@ $(EMCCFLAGS)

clean: $(addsuffix _clean,$(DIRS))
	rm -f *.bc

$(addsuffix _clean,$(DIRS)):
	$(MAKE) -C $(subst _clean,,$@) clean
