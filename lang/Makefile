all: $(patsubst %.json, ../www/lang/%.json, $(wildcard *.json))

../www/lang/%.json: %.json
	../node_modules/.bin/json-minify $< > $@

clean:
	rm -f ../www/css/*.{css,css.map}
