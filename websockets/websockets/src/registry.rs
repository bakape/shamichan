use super::client::Client;
use protocol::{payloads::AuthKey, util::SetMap};
use std::collections::{HashMap, HashSet};
use std::net::IpAddr;
use std::rc::Rc;
use std::sync::Mutex;

// Keeps state and feed subscription of all clients
#[derive(Default)]
pub struct Registry {
	clients: HashMap<u64, ClientDescriptor>,

	// Maps for quick lookup of client sets
	by_thread: SetMap<u64, u64>,
	by_ip: SetMap<IpAddr, u64>,
	by_key: SetMap<AuthKey, u64>,

	// Have not yet had their feed initialization messages sent.
	// Mapped by feed.
	need_init: SetMap<u64, u64>,
}

impl Registry {
	// Remove client's registration with a thread
	fn remove_from_thread(&mut self, client: u64, thread: Option<u64>) {
		if let Some(t) = thread {
			self.by_thread.remove(&t, &client);
			self.need_init.remove(&t, &client);
		}
	}
}

// Also start pulsar on first registry access
protocol::gen_global!(, , Registry);

// Stores client state that needs to be accessed by outer services along with
// a smart pointer to the client itself
struct ClientDescriptor {
	// Zero denotes thread catalog. Is unset (default) - before the first sync
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

// Remove client from registry
pub fn remove_client(id: u64) {
	write(|c| {
		if let Some(desc) = c.clients.remove(&id) {
			c.by_ip.remove(&desc.ip, &id);
			if let Some(key) = desc.key {
				c.by_key.remove(&key, &id);
			}
			c.remove_from_thread(id, desc.thread);
		}
	});
}

// Get a client by ID, if any
pub fn get_client(id: u64) -> Option<Rc<Mutex<super::client::Client>>> {
	// Release lock on global collection as soon as possible.
	read(|r| r.clients.get(&id).map(|c| c.client.clone()))
}

// Register a freshly created client with no messages received yet
pub fn add_client(id: u64, ip: IpAddr) {
	write(|c| {
		c.by_ip.insert(ip, id);
		c.clients.insert(id, ClientDescriptor::new(id, ip));
	});
}

// Set client auth key on first sync. Must only be done once per client.
pub fn set_client_key(id: u64, key: AuthKey) {
	write(|c| {
		if let Some(desc) = c.clients.get_mut(&id) {
			c.by_key.insert(key.clone(), id);
			desc.key = Some(key);
		}
	});
}

// Set or change the thread a client is synced to
pub fn set_client_thread(client: u64, thread: u64) {
	write(|c| {
		if let Some(desc) = c.clients.get_mut(&client) {
			let old = desc.thread;
			desc.thread = Some(thread);
			c.remove_from_thread(client, old);
			c.by_thread.insert(thread, client);
			c.need_init.insert(thread, client);
		}
	});
}

// Sync snapshot of client and thread data.
//
// Reads client that need to be initialized with drainer.
// Returns global set of connected clients and clients mapped by thread.
pub fn snapshot_threads<F>(drainer: F) -> (HashSet<u64>, SetMap<u64, u64>)
where
	F: FnOnce(&mut SetMap<u64, u64>),
{
	write(|c| {
		drainer(&mut c.need_init);
		(c.clients.keys().cloned().collect(), c.by_thread.clone())
	})
}
