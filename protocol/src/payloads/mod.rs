pub mod post_body;

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
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

// Define a public payload struct will public fields
macro_rules! payload {
    ($name:ident {$($field:ident: $t:ty,)*}) => {
        #[derive(Serialize, Deserialize, Default, Debug, Clone)]
        pub struct $name {
            $(pub $field: $t),*
        }
    }
}

// Authenticate with the server
payload! { Handshake {
	// Protocol version the client implements
	protocol_version: u16,

	// Used to authenticate the client
	key: AuthKey,
}}

// Request for creating a new thread
payload! { ThreadCreationReq {
	subject: String,
	tags: Vec<String>,
	captcha_solution: Vec<u8>,
}}

// Request for creating a new post
payload! { PostCreationReq {
	thread: u64,
	name: String,
	body: String,
}}

// State of an open post. Used to diff the current state of the client against
// the server feed's state.
payload! { OpenPost {
	has_image: bool,
	spoilered_image: bool,
	created_on: u64,
	thread: u64,
	body: post_body::Node,
}}

// Feed initialization data
payload! { FeedData {
	// Thread this feed refers to
	thread: u64,

	// Posts created in the last 16 minutes (open post CD + 1 minute to ensure
	// there is no overlap due to latency).
	// <post_id: post_creation_unix_timestamp>
	recent_posts: HashMap<u64,u64>,

	// Posts currently open. Mapped by ID.
	open_posts: HashMap<u64, OpenPost>,

	// TODO: Applied moderation
}}
