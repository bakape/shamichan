#[macro_export]
// Cast &str to C string, while keeping the same variable name.
// Needed to make sure the string is not dropped before the C  function returns.
macro_rules! to_C_string {
	( $var:ident, $fn:expr ) => (
		{
			let $var = ::std::ffi::CString::new($var).unwrap();
			{
				let $var = $var.as_ptr();
				$fn
			}
		}
	)
}
