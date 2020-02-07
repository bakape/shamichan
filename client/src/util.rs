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
	serde_json::error::Error,
	base64::DecodeError,
	std::io::Error,
	std::num::ParseIntError
}

// Shorthand for most commonly used Result type
pub type Result<T = ()> = std::result::Result<T, Error>;

// Generate functions for safely accessing a global variable
#[macro_export]
macro_rules! gen_global {
	($visibility:vis, $type:ty, $default:expr, $getter:ident) => {
		// Open global for writing
		#[allow(unused)]
		$visibility fn $getter() -> &'static mut $type  {
			unsafe {
				static mut GLOBAL: Option<$type> = None;
				if GLOBAL.is_none() {
					GLOBAL = Some($default);
				}
				GLOBAL.as_mut().unwrap()
			}
		}
	};
	($visibility:vis, $type:ty, $default:expr) => {
		$crate::gen_global!($visibility, $type, $default, get);
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

// Define function that caches global JS variable lookup
#[macro_export]
macro_rules! def_cached_getter {
	($visibility:vis, $name:ident, $type:ty, $get:expr) => {
		$visibility fn $name() -> $type {
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

// Log any error to both console and alert
pub fn log_error_res<T, E: Into<Error>>(res: std::result::Result<T, E>) {
	if let Err(err) = res {
		log_error(err.into());
	}
}

// Log error to both console and alert
pub fn log_error<T: Into<String>>(err: T) {
	let s = err.into();
	if s.len() != 0 {
		web_sys::console::error_1(&JsValue::from(&s));
	}
}
