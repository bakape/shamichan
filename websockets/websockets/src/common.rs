// Boxed error result type shorthand
pub type DynResult<T = ()> = Result<T, Box<dyn std::error::Error>>;

// Return a string as error
#[macro_export]
macro_rules! str_err {
	($msg:expr) => {
		return Err($msg.to_owned().into());
	};
	($fmt:expr, $( $args:tt )* ) => {
		str_err!(format!($fmt, $($args)*))
    };
}

// Run future within tokio runtime, blocking until it completes
pub fn run_future<F>(f: F) -> F::Output
where
	F: futures::prelude::Future,
{
	use tokio::runtime::Runtime;

	lazy_static! {
		static ref RUNTIME: Runtime = Runtime::new()
			.map_err(|e| format!("could not start tokio runtime: {}", e))
			.unwrap();
	}

	RUNTIME.handle().block_on(f)
}
