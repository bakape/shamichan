extern crate libc;

mod dom;

fn main() {
	dom::start();
	dom::set_inner_html("threads", "");
	dom::append("threads", "Hello world!");
}
