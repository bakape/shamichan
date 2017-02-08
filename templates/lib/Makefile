sserver:
	cargo rustc --release
	cp target/release/libminisin.a .

client: client_clean
	rustc --target wasm32-unknown-emscripten -v -O -o minisin.js src/lib.rs

client_clean:
	rm -f *.o *.js *.wasm *.wast

clean: client_clean
	rm -rf target *.a

debug:
	cargo rustc
