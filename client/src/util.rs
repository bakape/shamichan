use web_sys;

// Get JS window global
pub fn window() -> web_sys::Window {
	web_sys::window().expect("window undefined")
}

// Get page document
pub fn document() -> web_sys::Document {
	window().document().expect("document undefined")
}

// Get page body
pub fn body() -> web_sys::HtmlElement {
	document().body().expect("body undefined")
}

// Generate functions for safely accessing global variable.
//
// $type must be public
#[macro_export]
macro_rules! gen_global {
	($type:ty, $default:expr) => {
		// Open global for writing
		#[allow(unused)]
		pub fn with<F, R>(cb: F) -> R
		where
			F: FnOnce(&mut $type) -> R,
		{
			static ONCE: std::sync::Once = std::sync::Once::new();
			static mut GLOBAL: Option<$type> = None;
			ONCE.call_once(|| unsafe { GLOBAL = Some($default) });

			cb(unsafe { GLOBAL.as_mut().unwrap() })
		}
	};
	($type:ty) => {
		super::gen_global!($type, Default::default());
	};
}
