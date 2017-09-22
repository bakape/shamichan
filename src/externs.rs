#![allow(dead_code)] // TEMP

pub mod local_storage {
	use std::os::raw::c_char;

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
