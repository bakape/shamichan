all:
	npm install --progress false --depth 0
	npm run build
	mkdir -p ../www/client
	rm -rf ../www/client/*
	cp dist/* ../www/client/

install_tools:
	cargo install -f wasm-pack

watch:
	npm run start
