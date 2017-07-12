#![allow(dead_code)] // TEMP

mod mutations;
mod virtual_dom;

pub use self::mutations::*;
pub use self::virtual_dom::*;
use libc;

static mut ID_COUNTER: u64 = 0;

// Generate a new unique node ID
pub fn new_id() -> String {
	let s = format!("brunhild-{}", unsafe { ID_COUNTER });
	unsafe { ID_COUNTER += 1 };
	s
}

// Register flush_mutations() with emscripten event loop
pub fn start() {
	unsafe {
		emscripten_set_main_loop(mutations::flush_mutations, 0, 0);
	}
}

extern "C" {
	pub fn emscripten_set_main_loop(func: extern "C" fn(),
	                                fps: libc::c_int,
	                                infinite: libc::c_int);
}
