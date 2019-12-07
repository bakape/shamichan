use super::common;
use std::borrow::BorrowMut;
use std::sync::{Mutex, Once};

static INIT: Once = Once::new();
static mut PULSAR: Option<Mutex<Pulsar>> = None;

// TODO: Generate and pass feed state on client init
// TODO: Cache init state
// TODO: Dispatch updates every 100 ms
// TODO: Grab clients needing init from registry on pulse

#[derive(Default)]
struct Pulsar {
	init_msg_cache: Vec<u8>,
}

// Run function with obtained PULSAR access
fn with<F, R>(cb: F) -> R
where
	F: FnOnce(&mut Pulsar) -> R,
{
	unsafe { common::init_once(&INIT, &mut PULSAR) };
	cb(unsafe { PULSAR.as_ref().unwrap().lock().unwrap().borrow_mut() })
}
