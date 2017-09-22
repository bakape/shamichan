#[macro_use]
extern crate serde_derive;
#[macro_use]
extern crate brunhild;
extern crate serde_json;
extern crate serde;
extern crate libc;

#[macro_use]
mod externs;
mod posts;
pub mod state;
pub mod page;

fn main() {
	brunhild::start();
	// let data:  = serde_json::from_str(&dom::get_inner_html("post_data"));
	if let Err(e) = state::load() {
		println!("{}", e);
	}
}
