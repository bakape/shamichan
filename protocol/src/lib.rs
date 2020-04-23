mod codec;
mod message_types;
pub mod payloads;
pub mod util;

pub use codec::{Decoder, Encoder};
pub use message_types::MessageType;

#[macro_use]
extern crate num_derive;

// Version of protocol. Increment this on change.
pub const VERSION: u16 = 0;
