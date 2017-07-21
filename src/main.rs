#[macro_use]
extern crate serde_derive;
extern crate serde_json;
extern crate serde;
extern crate libc;

#[macro_use]
mod externs;
mod dom;
mod posts;
mod state;

fn main() {
	dom::start();
	state::load();
}
