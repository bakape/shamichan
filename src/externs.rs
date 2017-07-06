use std::ffi::CString;

pub fn alert(msg: &str) {
	// Forces the compiler to keep this pointer around, until the scope exits
	let _msg = CString::new(msg).unwrap();
	let __msg = _msg.as_ptr();
	unsafe { ffi::alert(__msg) };
}

macro_rules! pass_html {
	($id:expr, $html:expr, $fn:expr) => (
		let _id = CString::new($id).unwrap();
		let _html = CString::new($html).unwrap();
		let __id = _id.as_ptr();
		let __html = _html.as_ptr();
		unsafe { $fn(__id, __html) };
	)
}

pub fn set_outer_html(id: &str, html: &str) {
	pass_html!(id, html, ffi::set_outer_html);
}

pub fn set_inner_html(id: &str, html: &str) {
	pass_html!(id, html, ffi::set_inner_html);
}

pub fn pop_children(id: &str, n: i32) {
	let _id = CString::new(id).unwrap();
	let __id = _id.as_ptr();
	unsafe { ffi::pop_children(__id, n) }
}

pub fn append_element(id: &str, html: &str) {
	pass_html!(id, html, ffi::append_element);
}

pub mod ffi {
	use libc::*;

	extern "C" {
		pub fn alert(msg: *const c_char);
		pub fn set_outer_html(id: *const c_char, html: *const c_char);
		pub fn set_inner_html(id: *const c_char, html: *const c_char);
		pub fn append_element(id: *const c_char, html: *const c_char);
		pub fn pop_children(id: *const c_char, count: c_int);
		pub fn emscripten_set_main_loop(func: extern "C" fn(),
		                                fps: c_int,
		                                infinite: c_int);
	}
}
