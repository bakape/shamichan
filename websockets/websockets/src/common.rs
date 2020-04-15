// Boxed error result type shorthand
pub type DynResult<T = ()> = Result<T, Box<dyn std::error::Error>>;

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
