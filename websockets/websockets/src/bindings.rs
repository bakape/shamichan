use super::common::DynResult;
use super::{config, pulsar};
use libc;
use protocol::debug_log;
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

impl From<&[u8]> for WSBuffer {
	fn from(s: &[u8]) -> WSBuffer {
		Self {
			data: s.as_ptr(),
			size: s.len(),
		}
	}
}

// Like WSBuffer, but points to malloced data and frees itself on drop
#[repr(C)]
#[derive(Debug)]
pub struct WSBufferOwned {
	data: *mut u8,
	size: usize,
}

impl Default for WSBufferOwned {
	fn default() -> Self {
		Self {
			data: null_mut(),
			size: 0,
		}
	}
}

impl AsRef<[u8]> for WSBufferOwned {
	fn as_ref(&self) -> &[u8] {
		unsafe {
			std::slice::from_raw_parts(
				if self.data != null_mut() {
					self.data
				} else {
					std::ptr::NonNull::dangling().as_ptr()
				},
				self.size,
			)
		}
	}
}

impl Drop for WSBufferOwned {
	fn drop(&mut self) {
		unsafe { libc::free(self.data as *mut libc::c_void) };
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

	debug_log!("closing client", err);
	with_borrowed_c_char(err, |err| unsafe { ws_close_client(id, err) });
}

// Casts &str to borrowed C string and runs f() with it as an argument
fn with_borrowed_c_char<R>(s: &str, f: impl FnOnce(*const c_char) -> R) -> R {
	// Putting it on the stack here ensures this is not dropped until the call
	// to ws_close_client() returns
	let cs = CString::new(s).expect("null in Rust string");
	f(cs.as_ptr())
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
	subject: &str,
	tags: &[String],
	public_key: u64,
	body: &[u8],
) -> DynResult<u64> {
	let mut _tags: Vec<CString> = Vec::with_capacity(tags.len());
	for t in tags {
		_tags.push(CString::new(String::from(t))?);
	}
	let __tags: Vec<*const c_char> = _tags.iter().map(|x| x.as_ptr()).collect();

	let mut id: u64 = 0;
	cast_c_err(unsafe {
		ws_insert_thread(
			CString::new(subject)?.as_ptr(),
			__tags.as_ptr(),
			__tags.len(),
			public_key,
			body.into(),
			&mut id as *mut u64,
		)
	})?;
	Ok(id)
}

// Log error on Go side
pub fn log_error(err: &str) {
	with_borrowed_c_char(err, |err| unsafe { ws_log_error(err) });
}

// Propagate select configuration changes to Rust side
#[no_mangle]
extern "C" fn ws_set_config(buf: WSBuffer) -> *mut c_char {
	cast_to_c_error(|| -> Result<(), serde_json::Error> {
		let new = serde_json::from_slice(buf.as_ref())?;
		config::write(|c| *c = new);
		Ok(())
	})
}

// Initialize module
#[no_mangle]
extern "C" fn ws_init(feed_data: WSBuffer) -> *mut c_char {
	cast_to_c_error(|| -> Result<(), String> {
		pulsar::init(feed_data.as_ref())
			.map_err(|e| format!("could not start pulsar: {}", e))?;
		Ok(())
	})
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
			serde_json::from_slice::<protocol::payloads::ImageJSON>(
				image.as_ref(),
			)?
			.into(),
		)?;
		Ok(())
	})
}

// Register public key in the DB (if not already registered) and return its
// private ID, public ID and if the key was freshly registered
pub fn register_public_key(
	pub_key: &[u8],
) -> Result<(u64, uuid::Uuid, bool), String> {
	let mut priv_id = 0_u64;
	let mut pub_id: [u8; 16] = Default::default();
	let mut fresh = false;
	cast_c_err(unsafe {
		ws_register_public_key(
			pub_key.into(),
			&mut priv_id as *mut u64,
			&mut pub_id[0] as *mut u8,
			&mut fresh as *mut bool,
		)
	})?;
	Ok((priv_id, uuid::Uuid::from_bytes(pub_id), fresh))
}

// Get public key by its public ID together with its private ID
pub fn get_public_key(
	pub_id: uuid::Uuid,
) -> Result<(u64, WSBufferOwned), String> {
	let mut priv_id = 0_u64;
	let mut pub_key = WSBufferOwned::default();
	cast_c_err(unsafe {
		ws_get_public_key(
			pub_id.as_bytes().as_ptr(),
			&mut priv_id as *mut u64,
			&mut pub_key as *mut WSBufferOwned,
		)
	})?;
	Ok((priv_id, pub_key))
}

// Get thread and page numbers a post is in.
// Returns OK(None), if post does not exist.
pub fn get_post_parenthood(id: u64) -> Result<Option<(u64, u32)>, String> {
	let mut thread = 0;
	let mut page = 0;
	match cast_c_err(unsafe {
		ws_get_post_parenthood(
			id,
			&mut thread as *mut u64,
			&mut page as *mut u32,
		)
	}) {
		Ok(_) => Ok(Some((thread, page))),
		Err(err) => {
			if err == "no rows in result set" {
				Ok(None)
			} else {
				Err(err)
			}
		}
	}
}

// Increments spam detection score of a public key and sends captcha requests,
// if score exceeded.
pub fn increment_spam_score(pub_key: u64, score: usize) {
	unsafe {
		ws_increment_spam_score(pub_key, score as u64);
	}
}

// Write open post bodies to DB.
// Bodies must be in map[string]Body JSON map format.
pub fn write_open_post_bodies(buf: &[u8]) -> Result<(), String> {
	cast_c_err(unsafe { ws_write_open_post_bodies(buf.into()) })
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
		public_key: u64,
		body: WSBuffer,
		id: *mut u64,
	) -> *mut c_char;
	fn ws_register_public_key(
		pub_key: WSBuffer,
		priv_id: *mut u64,
		pub_id: *mut u8,
		fresh: *mut bool,
	) -> *mut c_char;
	fn ws_get_public_key(
		pub_id: *const u8,
		priv_id: *mut u64,
		pub_key: *mut WSBufferOwned,
	) -> *mut c_char;
	fn ws_get_post_parenthood(
		id: u64,
		thread: *mut u64,
		page: *mut u32,
	) -> *mut c_char;
	fn ws_increment_spam_score(pub_key: u64, score: u64);
	fn ws_write_open_post_bodies(buf: WSBuffer) -> *mut c_char;
}
