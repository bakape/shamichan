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

impl AuthKey {
	// Return pointer to inner array
	pub fn as_ptr(&self) -> *const u8 {
		&self.inner[0] as *const u8
	}
}

impl AsRef<[u8]> for AuthKey {
	fn as_ref(&self) -> &[u8] {
		&self.inner
	}
}

impl AsMut<[u8]> for AuthKey {
	fn as_mut(&mut self) -> &mut [u8] {
		&mut self.inner
	}
}

impl std::fmt::Debug for AuthKey {
	fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
		write!(f, "{:x}", self)
	}
}

impl std::fmt::LowerHex for AuthKey {
	fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
		for i in self.inner.iter() {
			write!(f, "{:x}", i)?;
		}
		Ok(())
	}
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
#[derive(Serialize, Deserialize, Debug)]
pub struct Handshake {
	// Protocol version the client implements
	pub protocol_version: u16,

	// Used to authenticate the client
	pub key: AuthKey,
}

// Request for creating a new thread
#[derive(Serialize, Deserialize, Debug)]
pub struct ThreadCreationReq {
	pub subject: String,
	pub tags: Vec<String>,
	pub captcha_solution: Vec<u8>,
}

// Request for creating a new post
#[derive(Serialize, Deserialize, Debug)]
pub struct PostCreationReq {
	pub thread: u64,
	pub name: String,
	pub body: String,
}

// Feed initialization data
#[derive(Serialize, Deserialize, Default, Debug)]
pub struct FeedData {
	pub feed: u64,
	// TODO: Data
}
