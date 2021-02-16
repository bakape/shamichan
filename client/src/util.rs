use std::fmt::Write;
use wasm_bindgen::prelude::JsValue;
use wasm_bindgen::prelude::*;
use wasm_bindgen::JsCast;
use web_sys;

/// Simple string error type for passing between subsystems and FFI
#[derive(Debug)]
pub struct Error(String);

impl Error {
	pub fn new(msg: String) -> Self {
		Self(msg)
	}
}

impl Into<JsValue> for Error {
	fn into(self) -> JsValue {
		JsValue::from(&self.0)
	}
}

impl Into<String> for Error {
	fn into(self) -> String {
		self.0
	}
}

impl AsRef<str> for Error {
	fn as_ref(&self) -> &str {
		&self.0
	}
}

impl From<JsValue> for Error {
	fn from(v: JsValue) -> Error {
		Error(format!("{:?}", v))
	}
}

impl std::fmt::Display for Error {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		write!(f, "{}", self.0)
	}
}

/// Trait specialization when?
macro_rules! from_display {
	($($type:ty),+) => {
		$(
			impl From<$type> for Error {
				fn from(err: $type) -> Error {
					Error(err.to_string())
				}
			}
		)+
	};
}
from_display! {
	String,
	&str,
	serde_json::error::Error,
	base64::DecodeError,
	std::io::Error,
	std::num::ParseIntError,
	anyhow::Error,
	bincode::Error,
	std::string::FromUtf8Error,
	std::fmt::Error
}

/// Shorthand for most commonly used Result type
pub type Result<T = ()> = std::result::Result<T, Error>;

/// Cache global JS variable lookup
#[macro_export]
macro_rules! cache_variable {
	($type:ty, $get:expr) => {{
		static mut CACHED: Option<$type> = None;
		unsafe {
			if CACHED.is_none() {
				CACHED = Some($get());
			}
			CACHED.as_ref().unwrap()
		}
	}};
}

/// Define function that caches global JS variable lookup
#[macro_export]
macro_rules! def_cached_getter {
	(
		$(#[$meta:meta])*
		$vis:vis $name:ident() -> $type:ty {
			$get:expr
		}
	) => {
		$(#[$meta])*
		$vis fn $name() -> &'static $type {
			$crate::cache_variable! { $type, || $get }
		}
	};
}

def_cached_getter! {
	/// Get JS window global
	pub window() -> web_sys::Window {
		web_sys::window().expect("window undefined")
	}
}

def_cached_getter! {
	/// Get page document
	pub document() -> web_sys::Document {
		window().document().expect("document undefined")
	}
}

def_cached_getter! {
	/// Get local storage manager
	pub local_storage() -> web_sys::Storage {
		window().local_storage().unwrap().unwrap()
	}
}

def_cached_getter! {
	/// Get the host part of the current location
	pub host() -> String {
		window().location().host().unwrap()
	}
}

/// Add static DOM event listener
pub fn add_static_listener<E>(
	target: &impl AsRef<web_sys::EventTarget>,
	event: &str,
	passive: bool,
	cb: yew::Callback<E>,
) where
	E: wasm_bindgen::convert::FromWasmAbi + 'static,
{
	add_listener(target, event, passive, cb).forget();
}

/// Add DOM event listener.
/// Returns created closure, that can be dropped to free resources.
pub fn add_listener<E>(
	target: &impl AsRef<web_sys::EventTarget>,
	event: &str,
	passive: bool,
	cb: yew::Callback<E>,
) -> Closure<dyn Fn(E)>
where
	E: wasm_bindgen::convert::FromWasmAbi + 'static,
{
	let cl = Closure::wrap(Box::new(move |e: E| cb.emit(e)) as Box<dyn Fn(E)>);
	target
		.as_ref()
		.add_event_listener_with_callback_and_add_event_listener_options(
			event,
			cl.as_ref().unchecked_ref(),
			&{
				let mut opts = web_sys::AddEventListenerOptions::new();
				opts.passive(passive);
				opts
			},
		)
		.unwrap_throw();
	cl
}

/// Log any error to console
pub fn log_error_res<T, E: Into<Error>>(res: std::result::Result<T, E>) {
	if let Err(err) = res {
		log_error(&err.into());
	}
}

/// Log error to console
pub fn log_error(err: &impl std::fmt::Display) {
	web_sys::console::error_1(&JsValue::from(err.to_string()));
}

/// Log a warning Message
#[allow(unused)]
pub fn log_warn(msg: impl AsRef<str>) {
	web_sys::console::warn_1(&msg.as_ref().into());
}

/// Display error alert message
pub fn alert(msg: &impl std::fmt::Display) {
	// Ignore result
	window().alert_with_message(&format!("error: {}", msg)).ok();
}

/// Log error to console and display it in an alert message
pub fn log_and_alert_error(err: &impl std::fmt::Display) {
	log_error(&err);
	alert(&err);
}

/// Run closure, logging any errors to both console error log and alert dialogs.
//
/// Returns default value in case of an error.
pub fn with_logging<T: Default>(f: impl FnOnce() -> Result<T>) -> T {
	match f() {
		Ok(v) => v,
		Err(e) => {
			log_and_alert_error(&e);
			Default::default()
		}
	}
}

/// Run async function, logging any errors to both console error log and alert
/// dialogs
pub async fn with_logging_async<R, A>(f: impl FnOnce(A) -> R, arg: A)
where
	R: futures::Future<Output = Result>,
{
	if let Err(e) = f(arg).await {
		log_error(&e);
	}
}

/// Format a duration into hours:mins:secs with padding and stripping headers,
/// as needed
pub fn format_duration(secs: impl Into<u64>) -> String {
	let secs_ = secs.into();
	let mut w = String::new();

	#[rustfmt::skip]
	macro_rules! write_bound {
		($bound:expr) => {
			if secs_ >= $bound {
				write!(&mut w, "{:0>2}:", secs_ / $bound).unwrap();
			}
		};
	}

	write_bound!(60 * 60);
	write_bound!(60);
	write!(&mut w, "{:0>2}", secs_ % 60).unwrap();

	w
}

/// Build a JS array from convertible item iterator
pub fn into_js_array<T, I>(it: I) -> js_sys::Array
where
	T: Into<JsValue>,
	I: IntoIterator<Item = T>,
{
	let arr = js_sys::Array::new();
	for i in it {
		arr.push(&i.into());
	}
	arr
}
