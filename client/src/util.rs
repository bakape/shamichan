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

impl From<JsValue> for Error {
	fn from(v: JsValue) -> Error {
		Error(format!("{:?}", v))
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
from_display!(
	serde_json::error::Error,
	base64::DecodeError,
	std::io::Error
);

// Shorthand for most commonly used Result type
pub type Result<T = ()> = std::result::Result<T, Error>;

// Generate functions for safely accessing a global variable
#[macro_export]
macro_rules! gen_global {
	($visibility:vis, $type:ty, $default:expr) => {
		// Open global for writing
		#[allow(unused)]
		$visibility fn with<'a, F, R>(mut cb: F) -> R
		where
			F: FnMut(&'a mut $type) -> R,
		{
			unsafe {
				static mut GLOBAL: Option<$type> = None;
				if GLOBAL.is_none() {
					GLOBAL = Some($default);
				}
				cb(unsafe { GLOBAL.as_mut().unwrap() })
			}
		}
	};
	($type:ty) => {
		$crate::gen_global!(, $type, Default::default());
	};
	($visibility:vis, $type:ty) => {
		$crate::gen_global!($visibility, $type, Default::default());
	};
	($type:ty, $default:expr) => {
		$crate::gen_global!(, $type, $default);
	};
}

// Wrap and cache static Rust callback closure
#[macro_export]
macro_rules! cache_cb {
	($type:ty, $fn:expr) => {
		unsafe {
			use wasm_bindgen::prelude::*;
			use wasm_bindgen::JsCast;

			static mut CACHED: Option<Closure<$type>> = None;
			if CACHED.is_none() {
				CACHED = Some(Closure::wrap(Box::from(&$fn)));
				}
			CACHED.as_ref().unwrap().as_ref().unchecked_ref()
			}
	};
}

// Cache element lookup by ID.
//
// Panics, if element not found.
#[macro_export]
macro_rules! cache_el {
	($id:expr) => {
		$crate::cache_variable! {
			web_sys::Element,
			|| match $crate::util::document().get_element_by_id($id) {
				Some(el) => el,
				None => panic!(format!("element not found: #{}", $id))
			}
		}
	};
}

// Cache global JS variable lookup
#[macro_export]
macro_rules! cache_variable {
	($type:ty, $get:expr) => {{
		static mut CACHED: Option<$type> = None;
		unsafe {
			if CACHED.is_none() {
				CACHED = Some($get());
				}
			CACHED.clone().unwrap()
			}
		}};
}

// Get JS window global
pub fn window() -> web_sys::Window {
	cache_variable! {
		web_sys::Window,
		|| web_sys::window().expect("window undefined")
	}
}

// Get page document
pub fn document() -> web_sys::Document {
	cache_variable! {
		web_sys::Document,
		|| window().document().expect("document undefined")
	}
}

// Get local storage manager
pub fn local_storage() -> web_sys::Storage {
	cache_variable! {
		web_sys::Storage,
		|| window().local_storage().unwrap().unwrap()
	}
}

// Wrap and cache static Rust callback closure as DOM event handler
//
// fn: handler with signature Fn(web_sys::Event) -> util::Result
#[macro_export]
macro_rules! event_handler {
	($fn:expr) => {{
		use web_sys;
		cache_cb!(dyn Fn(web_sys::Event), |e| {
			$crate::util::log_error_res($fn(e));
			})
		}};
}

// Add static passive DOM event listener.
//
// Use event_handler! to construct event handler.
pub fn add_listener<E>(target: E, typ: &str, handler: &js_sys::Function)
where
	E: JsCast,
{
	target
		.unchecked_ref::<web_sys::HtmlElement>()
		.add_event_listener_with_callback_and_add_event_listener_options(
			typ,
			handler,
			&{
				let mut opts = web_sys::AddEventListenerOptions::new();
				opts.passive(true);
				opts
			},
		)
		.unwrap();
}

// Log any error to both console and alert
pub fn log_error_res<T>(res: Result<T>) {
	if let Err(err) = res {
		log_error(err);
	}
}

// Log error to both console and alert
pub fn log_error<T: Into<String>>(err: T) {
	let s = err.into();
	if s.len() != 0 {
		window()
			.alert_with_message(&format!("error: {}", s))
			.expect("alert failed");
		web_sys::console::error_1(&JsValue::from(&s));
	}
}
