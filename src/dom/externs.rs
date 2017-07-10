use std::ffi::CString;

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
                prepend,
                before,
                after);

pub fn remove(id: &str) {
	to_C_string!(id, {
		unsafe { ffi::remove(id) };
	})
}

// Returns the inner HTML of an element by ID.
// If no element found, an empty String is returned.
// Usage of this function will cause extra repaints, so use sparingly.
#[allow(dead_code)]
pub fn get_inner_html(id: &str) -> String {
	to_C_string!(id, {
		from_C_string!(ffi::get_inner_html(id))
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
	                prepend,
	                before,
	                after);

	extern "C" {
		pub fn remove(id: *const c_char);
		pub fn get_inner_html(id: *const c_char) -> *mut c_char;
	}
}
