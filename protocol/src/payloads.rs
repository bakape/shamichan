use serde::{Deserialize, Serialize};
use std::hash::{Hash, Hasher};

// Helper for big array serialization
big_array! { BigArray; }

// Client authentication key type
#[derive(Serialize, Deserialize, Clone)]
pub struct AuthKey {
	#[serde(with = "BigArray")]
	inner: [u8; 64],
}

impl Hash for AuthKey {
	fn hash<H: Hasher>(&self, state: &mut H) {
		(&self.inner).hash(state);
	}
}

impl PartialEq for AuthKey {
	fn eq(&self, other: &AuthKey) -> bool {
		(&self.inner) as &[u8] == (&other.inner) as &[u8]
	}
}

impl Eq for AuthKey {}

impl Default for AuthKey {
	fn default() -> Self {
		Self { inner: [0; 64] }
	}
}

// Authenticate with the server
#[derive(Serialize, Deserialize)]
pub struct Handshake {
	// Protocol version the client implements
	pub protocol_version: u16,

	// Used to authenticate the client
	pub key: AuthKey,
}

// Request to synchronize with a specific thread on the server
#[derive(Serialize, Deserialize)]
pub struct SyncRequest {
	// Thread ID. 0 denotes global thread index.
	pub thread: u64,
}
