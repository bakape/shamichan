#[macro_use]
extern crate serde_derive;
extern crate serde_json;
extern crate serde;
extern crate libc;
extern crate url;

#[macro_use]
mod externs;
mod dom;
mod posts;
mod state;

fn main() {
	dom::start();
	state::load();

	let _: posts::Board =
		serde_json::from_str(&dom::get_inner_html("post-data")).unwrap();
}
