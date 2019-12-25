use super::client::Client;
use protocol::AuthKey;
use std::collections::{HashMap, HashSet};
use std::hash::Hash;
use std::net::IpAddr;
use std::rc::Rc;
use std::sync::Mutex;

// Keeps state and feed subscription of all clients
#[derive(Default)]
pub struct Registry {
	by_id: HashMap<u64, ClientDescriptor>,

	// Maps for quick lookup of client sets
	by_thread: SetMap<u64>,
	by_ip: SetMap<IpAddr>,
	by_key: SetMap<AuthKey>,

	// Have not yet had their feed initialization messages sent.
	// Mapped by thread.
	need_init: SetMap<u64>,
}

impl Registry {
	// Remove client's registration with a thread
	fn remove_from_thread(&mut self, client: u64, thread: Option<u64>) {
		if let Some(t) = thread {
			self.by_thread.remove(&t, client);
			self.need_init.remove(&t, client);
		}
	}
}

super::gen_global_rwlock!(Registry);

// Stores client state that needs to be accessed by outer services along with
// a smart pointer to the client itself
struct ClientDescriptor {
	// Zero denotes thread catalog. Is unset by default - before the first sync
	// message is received.
	thread: Option<u64>,

	key: Option<AuthKey>,
	ip: IpAddr,
	client: Rc<Mutex<Client>>,
}

impl ClientDescriptor {
	fn new(id: u64, ip: IpAddr) -> Self {
		Self {
			ip: ip,
			thread: None,
			key: None,
			client: Rc::new(Mutex::new(Client::new(id, ip))),
		}
	}
}

// Map of K to sets of client IDs
struct SetMap<K: Hash + Eq>(HashMap<K, HashSet<u64>>);

impl<K: Hash + Eq> Default for SetMap<K> {
	fn default() -> Self {
		Self(HashMap::new())
	}
}

impl<K: Hash + Eq + Clone> SetMap<K> {
	fn insert(&mut self, k: &K, client: u64) {
		match self.0.get_mut(k) {
			Some(set) => {
				set.insert(client);
			}
			None => {
				let mut set = HashSet::new();
				set.insert(client);
				self.0.insert(k.clone(), set);
			}
		}
	}

	fn remove<'k, 's: 'k>(
		&'s mut self,
		k: impl Into<Option<&'k K>>,
		client: u64,
	) {
		if let Some(k) = k.into() {
			if let Some(set) = self.0.get_mut(k) {
				set.remove(&client);
				if set.len() == 0 {
					self.0.remove(k);
				}
			}
		}
	}
}

// Remove client from registry
pub fn remove_client(id: u64) {
	write(|c| {
		if let Some(desc) = c.by_id.remove(&id) {
			c.by_ip.remove(&desc.ip, id);
			c.by_key.remove(&desc.key, id);
			c.remove_from_thread(id, desc.thread);
		}
	});
}

// Get a client by ID, if any
pub fn get_client(id: u64) -> Option<Rc<Mutex<super::client::Client>>> {
	// Release lock on global collection as soon as possible.
	read(|r| r.by_id.get(&id).map(|c| c.client.clone()))
}

// Register a freshly created client with no messages received yet
pub fn add_client(id: u64, ip: IpAddr) {
	write(|c| {
		c.by_ip.insert(&ip, id);
		c.by_id.insert(id, ClientDescriptor::new(id, ip));
	});
}

// Set client auth key on first sync. Must only be done once per client.
pub fn set_client_key(id: u64, key: AuthKey) {
	write(|c| {
		if let Some(desc) = c.by_id.get_mut(&id) {
			c.by_key.insert(&key, id);
			desc.key = Some(key);
		}
	});
}

// Set or change the thread a client is synced to
pub fn set_client_thread(client: u64, thread: u64) {
	write(|c| {
		if let Some(desc) = c.by_id.get_mut(&client) {
			let old = desc.thread;
			desc.thread = Some(thread);
			c.remove_from_thread(client, old);
			c.by_thread.insert(&thread, client);
			c.need_init.insert(&thread, client);
		}
	});
}
