use std::sync::Once;

// Boxed error result type shorthand
pub type DynResult<T = ()> = Result<T, Box<dyn std::error::Error>>;

// Initialize a value once on runtime
pub unsafe fn init_once<T: Default>(once: &Once, val: &mut Option<T>) {
	once.call_once(|| *val = Some(Default::default()));
}

// Return a string as error
#[macro_export]
macro_rules! str_err {
	($msg:expr) => {
		return Err($msg.into());
	};
	($fmt:expr, $( $args:tt )* ) => {
		str_err!(format!($fmt, $($args)*))
    };
}
