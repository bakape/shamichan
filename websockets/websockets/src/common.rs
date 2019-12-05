// Boxed error result type shorthand
pub type DynResult<T = ()> = Result<T, Box<dyn std::error::Error>>;
