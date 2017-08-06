#[macro_use]
extern crate serde_derive;
extern crate serde_json;
extern crate serde;
extern crate libc;

#[macro_use]
mod externs;
mod dom;
mod posts;
pub mod state;
pub mod page;

fn main() {
	dom::start();
	// let data:  = serde_json::from_str(&dom::get_inner_html("post_data"));
	if let Err(e) = state::load() {
		println!("{}", e);
	}
}
