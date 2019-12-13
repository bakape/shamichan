all: $(patsubst %.less, ../www/css/%.css, $(wildcard *.less))

../www/css/%.css: %.less
	../node_modules/.bin/lessc $< --source-map $@ --clean-css > $@

clean:
	rm -f ../www/css/*.{css,css.map}
