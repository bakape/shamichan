#! /bin/bash
set -e

nice -n 19 cargo build --release

cd client
nice -n 19 ./node_modules/.bin/webpack

cd ..
rm -r \
	client/src websockets/websockets/src protocol/src \
	target/release/deps/libwebsockets* \
	target/release/deps/libclient* \
	target/release/deps/libprotocol* \
	target/wasm32-unknown-unknown/release/deps/libprotocol* \
	client/dist client/pkg
