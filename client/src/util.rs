use std::fmt::Write;
use wasm_bindgen::prelude::JsValue;
use wasm_bindgen::JsCast;
use web_sys;

// Simple string error type for passing between subsystems and FFI
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

// Trait specialization when?
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
	std::string::FromUtf8Error
}

// Shorthand for most commonly used Result type
pub type Result<T = ()> = std::result::Result<T, Error>;

// Cache global JS variable lookup
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

// Define function that caches global JS variable lookup
#[macro_export]
macro_rules! def_cached_getter {
	($visibility:vis, $name:ident, $type:ty, $get:expr) => {
		$visibility fn $name() -> &'static $type {
			$crate::cache_variable! { $type, $get }
		}
	};
	($name:ident, $type:ty, $get:expr) => {
		def_cached_getter! { , $name,$type, $expr }
	};
}

// Get JS window global
def_cached_getter! { pub, window, web_sys::Window,
	|| web_sys::window().expect("window undefined")
}

// Get page document
def_cached_getter! { pub, document, web_sys::Document,
	|| window().document().expect("document undefined")
}

// Get local storage manager
def_cached_getter! { pub, local_storage, web_sys::Storage,
	|| window().local_storage().unwrap().unwrap()
}

// Get the host part of the current location
def_cached_getter! { pub, host, String,
	|| window().location().host().unwrap()
}

// Add static passive DOM event listener
pub fn add_static_listener<E>(
	target: &impl AsRef<web_sys::EventTarget>,
	event: &str,
	cb: yew::Callback<E>,
) where
	E: wasm_bindgen::convert::FromWasmAbi + 'static,
{
	use wasm_bindgen::prelude::*;

	let cl = Closure::wrap(Box::new(move |e: E| cb.emit(e)) as Box<dyn Fn(E)>);
	target
		.as_ref()
		.add_event_listener_with_callback_and_add_event_listener_options(
			event,
			cl.as_ref().unchecked_ref(),
			&{
				let mut opts = web_sys::AddEventListenerOptions::new();
				opts.passive(true);
				opts
			},
		)
		.unwrap();

	// Never drop the closure as this event handler is static
	cl.forget();
}

// Log any error to console
pub fn log_error_res<T, E: Into<Error>>(res: std::result::Result<T, E>) {
	if let Err(err) = res {
		log_error(err.into());
	}
}

// Log error to console
pub fn log_error<T: std::fmt::Display>(err: T) {
	web_sys::console::error_1(&JsValue::from(err.to_string()));
}

// Run closure, logging any errors to both console error log and alert dialogs
pub fn with_logging(f: impl FnOnce() -> Result) {
	if let Err(e) = f() {
		alert(&e);
		log_error(e);
	}
}

// Run async function, logging any errors to both console error log and alert
// dialogs
pub async fn with_logging_async<R, A>(f: impl FnOnce(A) -> R, arg: A)
where
	R: futures::Future<Output = Result>,
{
	if let Err(e) = f(arg).await {
		alert(&e);
		log_error(e);
	}
}

// Display error alert message
pub fn alert(msg: &impl std::fmt::Display) {
	// Ignore result
	window().alert_with_message(&format!("error: {}", msg)).ok();
}

// Format a duration into hours:mins:secs with padding and stripping headers,
// as needed
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

// Build a JS array from convertible item iterator
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

// Generate partial Component implementation for self.props as Self::Properties
// updates
#[macro_export]
macro_rules! comp_prop_change {
	($props:ty) => {
		type Properties = $props;

		fn change(&mut self, props: Self::Properties) -> bool {
			if self.props != props {
				self.props = props;
				true
			} else {
				false
			}
		}
	};
}

// Generate partial Component implementation for components with no possible
// property changes
#[macro_export]
macro_rules! comp_no_prop_change {
	($props:ty) => {
		type Properties = $props;

		fn change(&mut self, _: Self::Properties) -> bool {
			false
		}
	};
}

// Generate partial Component implementation with no properties
#[macro_export]
macro_rules! comp_no_props {
	() => {
		type Properties = ();

		fn change(&mut self, _: Self::Properties) -> bool {
			false
		}
	};
}

// Generate partial Component implementation for components with only a
// rerender message
#[macro_export]
macro_rules! comp_message_rerender {
	() => {
		type Message = ();

		fn update(&mut self, _: Self::Message) -> bool {
			true
		}
	};
}

// Generate Component update methods for static components
#[macro_export]
macro_rules! comp_static {
	($props:ty) => {
		$crate::comp_no_prop_change! {$props}
		type Message = ();

		fn update(&mut self, _: Self::Message) -> bool {
			false
		}
	};
	() => {
		$crate::comp_static! {()}
	};
}
