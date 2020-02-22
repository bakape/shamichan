all: reset
ifneq ($(NO_DEPS),1)
	cargo install wasm-pack
	npm install --progress false --depth 0
endif
	./node_modules/.bin/webpack $(if $(filter 1,$(DEBUG)),-d,)
	mkdir -p ../www/client
	rm -rf ../www/client/*
	cp dist/* ../www/client/

reset:
	rm -rf dist pkg
	mkdir pkg dist

watch: reset
	webpack-dev-server --open -d

clean:
	cargo clean
	rm -rf target dist pkg node_modules
