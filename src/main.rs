extern crate libc;

#[macro_use]
mod externs;
mod dom;

fn main() {
	dom::start();

	println!("{}", dom::get_inner_html("post-data"));
}
