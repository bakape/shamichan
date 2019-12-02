use std::borrow::{Borrow, BorrowMut};
use std::collections::HashMap;
use std::rc::Rc;
use std::sync::{Mutex, Once, RwLock};

// TODO: Thread subscriber map to ID set
// TODO: IP->ID set map

static INIT: Once = Once::new();
static mut CLIENTS: Option<RwLock<Clients>> = None;

// Shorthand.
//
// Client is wrapped in Mutex to guard against modifications of client internal
// state only. Any global collections should never acquire this lock.
//
// However, accessing global collections from within a help mutex client is
// permitted. This ensures a mutex acquiring order and thus prevents deadlocks.
pub type Clients = HashMap<u64, Rc<Mutex<super::client::Client>>>;

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
