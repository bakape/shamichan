use std::borrow::{Borrow, BorrowMut};
use std::collections::HashMap;
use std::sync::{Once, RwLock};

static INIT: Once = Once::new();
static mut CLIENTS: Option<RwLock<Clients>> = None;

// Maps to a websocket client on the Go side
pub struct Client {
	pub id: u64,
	pub ip: std::net::IpAddr,
}

// Shorthand
pub type Clients = HashMap<u64, Client>;

#[inline]
fn init() {
	INIT.call_once(|| unsafe { CLIENTS = Some(Default::default()) });
}

// Open registered client map for reading
#[inline]
pub fn read<F, R>(cb: F) -> R
where
	F: FnOnce(&Clients) -> R,
{
	init();
	cb(unsafe { CLIENTS.as_ref().unwrap().read().unwrap().borrow() })
}

// Open registered client map for writing
#[inline]
pub fn write<F, R>(cb: F) -> R
where
	F: FnOnce(&mut Clients) -> R,
{
	init();
	cb(unsafe { CLIENTS.as_ref().unwrap().write().unwrap().borrow_mut() })
}

// Remove client from collection
pub fn remove_client(id: u64) {
	write(|c| c.remove(&id));
}
