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

// Generate functions for safely accessing global variable behind a RWLock.
//
// $vis: accessor visibility
// $type: type to store; must implement Default
// $extra_init: extra initialization lambda to execute
#[macro_export]
macro_rules! gen_global_rwlock {
	($vis:vis, $type:ident, $extra_init:expr) => {
		static __ONCE: std::sync::Once = std::sync::Once::new();
		static mut __GLOBAL: Option<std::sync::RwLock<$type>> = None;

		fn __init() {
			__ONCE.call_once(|| {
				unsafe { __GLOBAL = Some(Default::default()) };
				$extra_init();
			});
		}

		// Open global for reading
		#[allow(unused)]
		$vis fn read<F, R>(cb: F) -> R
		where
			F: FnOnce(&$type) -> R,
		{
			__init();
			cb(&*unsafe { __GLOBAL.as_ref().unwrap().read().unwrap() })
		}

		// Open global for writing
		#[allow(unused)]
		$vis fn write<F, R>(cb: F) -> R
		where
			F: FnOnce(&mut $type) -> R,
		{
			__init();
			cb(&mut *unsafe { __GLOBAL.as_ref().unwrap().write().unwrap() })
		}
	};
	($type:ident) => {
		$crate::gen_global_rwlock!(,$type, || {});
	};
	($vis:vis, $type:ident) => {
		$crate::gen_global_rwlock!($vis, $type, || {});
	};
}
