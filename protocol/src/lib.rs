mod payloads;
pub use payloads::post_body;
pub use payloads::*;
mod util;
pub use util::*;
mod codec;
pub use codec::*;
mod message_types;
pub use message_types::*;

#[macro_use]
extern crate num_derive;
#[macro_use]
extern crate serde_big_array;

// Version of protocol. Increment this on change.
pub const VERSION: u16 = 0;
