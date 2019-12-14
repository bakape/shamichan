lazy_static! {
	pub static ref STATE: State = Default::default();
}

// Global state singleton
#[derive(Default)]
pub struct State {}
