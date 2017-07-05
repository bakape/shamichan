extern crate libc;

mod externs;
mod view;

fn main() {
	externs::alert("Hello world!");
}
