#![allow(dead_code)] // TEMP

#[macro_export]
// Cast &str to C string, while keeping the same variable name.
// Needed to make sure the string is not dropped before the C function returns.
macro_rules! to_C_string {
	( $var:ident, $fn:expr ) => (
		{
			let $var = ::std::ffi::CString::new($var).unwrap();
			{
				let $var = $var.as_ptr();
				$fn
			}
		}
	)
}

// Casts owned C string to String
#[macro_export]
macro_rules! from_C_string {
	($s:expr) => (
		unsafe { ::std::ffi::CString::from_raw($s) }
			.into_string()
			.unwrap()
	 )
}

pub mod local_storage {
	use libc::*;

	pub fn set(key: &str, val: &str) {
		to_C_string!(key, {
			to_C_string!(val, {
				unsafe { local_storage_set(key, val) };
			})
		})
	}

	pub fn get(key: &str) -> String {
		to_C_string!(key, {
			from_C_string!(local_storage_get(key))
		})
	}

	extern "C" {
		fn local_storage_set(key: *const c_char, val: *const c_char);
		fn local_storage_get(key: *const c_char) -> *mut c_char;
	}
}
