use libc;
use std::borrow::Cow;
use std::ffi::CStr;
use std::os::raw::c_char;
use std::ptr::null_mut;
use std::rc::Rc;

// Wrapper for passing buffer references over the FFI
#[repr(C)]
pub struct WSBuffer {
	data: *const u8,
	size: usize,
}

impl AsRef<[u8]> for WSBuffer {
	fn as_ref(&self) -> &[u8] {
		unsafe { std::slice::from_raw_parts(self.data, self.size) }
	}
}

// Like WSBuffer, but with pointer for reference counting on Rust side
#[repr(C)]
pub struct WSRcBuffer {
	inner: WSBuffer,
	src: *const Vec<u8>,
}

impl From<Rc<Vec<u8>>> for WSRcBuffer {
	fn from(src: Rc<Vec<u8>>) -> WSRcBuffer {
		Self {
			inner: WSBuffer {
				data: src.as_ptr(),
				size: src.len(),
			},
			src: Rc::into_raw(src),
		}
	}
}

// Register a websocket client with a unique ID and return any error
#[no_mangle]
extern "C" fn ws_register_client(id: u64, ip: WSBuffer) -> *mut c_char {
	// Wrapper to enable usage of ?
	fn wrapped(id: u64, ip: WSBuffer) -> Result<(), String> {
		super::registry::add_client(
			id,
			std::str::from_utf8(ip.as_ref())
				.map_err(|err| format!("could not read IP string: {}", err))?
				.parse()
				.map_err(|err| format!("{}", err))?,
		);
		Ok(())
	}

	cast_error(wrapped(id, ip))
}

// Cast result and allocate error message as owned C string, if any
fn cast_error<T>(r: Result<T, String>) -> *mut c_char {
	match r {
		Ok(_) => null_mut(),
		Err(err) => alloc_error(&err),
	}
}

// Allocate error message as owned C string, if any
fn alloc_error(err: &str) -> *mut c_char {
	let size = err.len();
	if size == 0 {
		null_mut()
	} else {
		unsafe {
			let buf = libc::malloc(size + 1) as *mut c_char;
			std::ptr::copy_nonoverlapping(
				err.as_ptr() as *const c_char,
				buf,
				size,
			);
			*buf.offset(size as isize) = 0;
			buf
		}
	}
}

// Pass received message to Rust side. This operation never returns an error to
// simplify error propagation. All errors are propagated back to Go only using
// ws_close_client.
#[no_mangle]
extern "C" fn ws_receive_message(client_id: u64, msg: WSBuffer) {
	// Client could be not found due to a race between the main client
	// goroutine and the reading goroutine.
	//
	// It's fine - unregistration can be eventual.
	if let Some(c) = super::registry::get_client(client_id) {
		if let Err(err) = c.lock().unwrap().receive_message(msg.as_ref()) {
			close_client(client_id, &err.to_string());
		}
	}
}

// Remove client from registry
#[no_mangle]
extern "C" fn ws_unregister_client(id: u64) {
	super::registry::remove_client(id);
}

// Unref and potentially free a message source on the Rust side
#[no_mangle]
extern "C" fn ws_unref_message(src: *const Vec<u8>) {
	unsafe { Rc::<Vec<u8>>::from_raw(src) }; // Drop it
}

// Send close message with optional error to client and unregister it
pub fn close_client(id: u64, err: &str) {
	// Go would still unregister the client eventually, but removing it early
	// will prevent any further message write attempts to it.
	super::registry::remove_client(id);

	unsafe { ws_close_client(id, alloc_error(err)) };
}

// Check, if thread exists in DB
pub fn thread_exists(id: u64) -> Result<bool, String> {
	let mut err: *mut c_char = null_mut();
	let exists = unsafe { ws_thread_exists(id, &mut err as *mut *mut c_char) };
	if err != null_mut() {
		let s: String = match unsafe { CStr::from_ptr(err) }.to_string_lossy() {
			Cow::Borrowed(e) => e.into(),
			Cow::Owned(e) => e,
		};
		unsafe { libc::free(err as *mut libc::c_void) };
		return Err(s);
	}
	return Ok(exists);
}

// Write message to specific client
pub fn write_message(client_id: u64, msg: Rc<Vec<u8>>) {
	unsafe { ws_write_message(client_id, msg.into()) };
}

// Linked from Go
extern "C" {
	fn ws_write_message(client_id: u64, msg: WSRcBuffer);
	fn ws_close_client(clientID: u64, err: *const c_char);
	fn ws_thread_exists(id: u64, er: *mut *mut c_char) -> bool;
}
