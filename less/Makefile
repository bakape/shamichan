all: $(patsubst %.less, ../www/css/%.css, $(wildcard *.less))

../www/css/%.css: %.less
	../node_modules/.bin/lessc $< \
		--source-map $@ \
		--clean-css \
		--source-map-rootpath=/assets/css \
		--source-map-less-inline \
		> $@

clean:
	rm -fr ../www/css/*.{css,css.map} ../www/css/maps
