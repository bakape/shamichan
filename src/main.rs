#[macro_use]
extern crate serde_derive;
extern crate serde_json;
extern crate serde;
extern crate libc;

#[macro_use]
mod externs;
mod dom;
mod posts;
mod options;

fn main() {
	dom::start();

	let json: posts::Board = serde_json::from_str(&dom::get_inner_html("post-data",),)
		.unwrap();
}
