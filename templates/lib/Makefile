server:
	cargo rustc --release
	cp target/release/libminisin.a .

client:
	rustc --target wasm32-unknown-emscripten --crate-type bin -v -O -o minisin.js src/lib.rs

clean:
	rm -rf target *.a *.o *.js *.wasm *.wast

debug:
	cargo rustc
