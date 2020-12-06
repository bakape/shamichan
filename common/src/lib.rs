mod codec;
pub mod config;
mod message_types;
pub mod payloads;
pub mod util;

pub use codec::{Decoder, Encoder};
pub use message_types::MessageType;

#[macro_use]
extern crate num_derive;
#[macro_use]
extern crate serde_big_array;

/// Version of common. Increment this on change.
pub const VERSION: u16 = 1;
