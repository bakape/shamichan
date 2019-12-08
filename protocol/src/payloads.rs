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

// Request for creating a new thread
#[derive(Serialize, Deserialize)]
pub struct ThreadCreationReq {
	pub subject: String,
	pub tags: Vec<String>,
}

// Request for creating a new post
#[derive(Serialize, Deserialize)]
pub struct PostCreationReq {
	pub thread: u64,
	pub name: String,
	pub body: String,
}
