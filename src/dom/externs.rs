use std::ffi::CString;

// Cast &str to C string, while keeping the same variable name.
// Needed to make sure the string is not dropped before the C  function returns.
macro_rules! to_C_string {
	( $var:ident, $fn:expr ) => (
		{
			let $var = CString::new($var).unwrap();
			{
				let $var = $var.as_ptr();
				$fn
			}
		}
	)
}

// Define functions for writing to the DOM
macro_rules! define_writers {
	( $( $id:ident ),* ) => (
		$(
			#[allow(dead_code)]
			pub fn $id(id: &str, html: &str) {
				to_C_string!(id, {
					to_C_string!(html, {
						unsafe { ffi::$id(id, html) };
					})
				})
			}
		)*
	 )
}

define_writers!(set_outer_html,
                set_inner_html,
                append,
                append_by_selector,
                prepend,
                before,
                after);

pub fn remove(id: &str) {
	to_C_string!(id, {
		unsafe { ffi::remove(id) };
	})
}

mod ffi {
	use libc::*;

	// Define external functions for writing to the DOM
	macro_rules! define_writers {
		( $( $id:ident ),* ) => (
			extern "C" {
				$( pub fn $id(id: *const c_char, html: *const c_char); )*
			}
		)
	}

	define_writers!(set_outer_html,
	                set_inner_html,
	                append,
	                append_by_selector,
	                prepend,
	                before,
	                after);

	extern "C" {
		pub fn remove(id: *const c_char);
	}
}
