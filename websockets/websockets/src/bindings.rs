use super::common::DynResult;
use super::{config, pulsar};
use libc;
use std::borrow::Cow;
use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_void};
use std::ptr::null_mut;
use std::sync::Arc;

// Wrapper for passing buffer references over the FFI
#[repr(C)]
#[derive(Debug)]
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
#[derive(Debug)]
pub struct WSRcBuffer {
	inner: WSBuffer,
	src: *const c_void,
}

impl From<Arc<Vec<u8>>> for WSRcBuffer {
	fn from(src: Arc<Vec<u8>>) -> WSRcBuffer {
		Self {
			inner: WSBuffer {
				data: src.as_ptr(),
				size: src.len(),
			},
			src: Arc::into_raw(src) as *const c_void,
		}
	}
}

// Register a websocket client with a unique ID and return any error
#[no_mangle]
extern "C" fn ws_register_client(id: u64, ip: WSBuffer) -> *mut c_char {
	cast_to_c_error(|| -> Result<(), String> {
		super::registry::add_client(
			id,
			std::str::from_utf8(ip.as_ref())
				.map_err(|err| format!("could not read IP string: {}", err))?
				.parse()
				.map_err(|err| format!("{}", err))?,
		);
		Ok(())
	})
}

// Cast error to owned C error and return it, if any
fn cast_to_c_error<E, F>(f: F) -> *mut c_char
where
	E: std::fmt::Display,
	F: FnOnce() -> Result<(), E>,
{
	match f() {
		Ok(_) => null_mut(),
		Err(src) => {
			let err = src.to_string();
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
extern "C" fn ws_unref_message(src: *const c_void) {
	unsafe { Arc::<Vec<u8>>::from_raw(src as *const Vec<u8>) }; // Drop it
}

// Send close message with optional error to client and unregister it
pub fn close_client(id: u64, err: &str) {
	// Go would still unregister the client eventually, but removing it early
	// will prevent any further message write attempts to it.
	super::registry::remove_client(id);

	unsafe { ws_close_client(id, CString::new(err).unwrap().as_ptr()) };
}

// Check, if thread exists in DB
pub fn thread_exists(id: u64) -> Result<bool, String> {
	let mut exists = false;
	cast_c_err(unsafe { ws_thread_exists(id, &mut exists as *mut bool) })?;
	return Ok(exists);
}

// Cast owned C error to Result
fn cast_c_err(err: *mut c_char) -> Result<(), String> {
	if err != null_mut() {
		let s: String = match unsafe { CStr::from_ptr(err) }.to_string_lossy() {
			Cow::Borrowed(e) => e.into(),
			Cow::Owned(e) => e,
		};
		unsafe { libc::free(err as *mut libc::c_void) };
		return Err(s);
	}
	Ok(())
}

// Write message to specific client
pub fn write_message(client_id: u64, msg: Arc<Vec<u8>>) {
	unsafe { ws_write_message(client_id, msg.into()) };
}

// Create a new thread and return it's ID
pub fn insert_thread(
	subject: String,
	tags: Vec<String>,
	auth_key: &protocol::AuthKey,
) -> DynResult<u64> {
	let mut _tags: Vec<CString> = Vec::with_capacity(tags.len());
	for t in tags {
		_tags.push(CString::new(t)?);
	}
	let __tags: Vec<*const c_char> = _tags.iter().map(|x| x.as_ptr()).collect();

	let mut id: u64 = 0;
	cast_c_err(unsafe {
		ws_insert_thread(
			CString::new(subject)?.as_ptr(),
			__tags.as_ptr(),
			__tags.len(),
			auth_key.as_ptr(),
			&mut id as *mut u64,
		)
	})?;
	Ok(id)
}

// Log error on Go side
pub fn log_error(err: &str) {
	unsafe { ws_log_error(CString::new(err).unwrap().as_ptr()) };
}

#[repr(C)]
struct WSConfig {
	captcha: bool,
}

// Pointless for now, but will add properties with some conversion needed later
impl Into<config::Config> for WSConfig {
	fn into(self) -> config::Config {
		config::Config {
			captcha: self.captcha,
		}
	}
}

// Propagate select configuration changes to Rust side
#[no_mangle]
extern "C" fn ws_set_config(wsc: WSConfig) {
	config::write(|c| *c = wsc.into());
}

// Initialize module
#[no_mangle]
extern "C" fn ws_init(feed_data: WSBuffer) {
	if let Err(err) = pulsar::init(feed_data.as_ref()) {
		panic!(format!("could not start pulsar: {}", err));
	}
}

// Register image insertion into an open post.
//
// image: JSON-encoded inserted image data
#[no_mangle]
extern "C" fn ws_insert_image(
	thread: u64,
	post: u64,
	image: WSBuffer,
) -> *mut c_char {
	cast_to_c_error(|| -> DynResult {
		pulsar::insert_image(
			thread,
			post,
			serde_json::from_slice::<protocol::ImageJSON>(image.as_ref())?
				.into(),
		)?;
		Ok(())
	})
}

// Linked from Go
extern "C" {
	fn ws_write_message(client_id: u64, msg: WSRcBuffer);
	fn ws_close_client(clientID: u64, err: *const c_char);
	fn ws_thread_exists(id: u64, exists: *mut bool) -> *mut c_char;
	fn ws_log_error(err: *const c_char);
	fn ws_insert_thread(
		subject: *const c_char,
		tags: *const *const c_char,
		tags_size: usize,
		auth_key: *const u8,
		id: *mut u64,
	) -> *mut c_char;
}
