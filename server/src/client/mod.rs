mod client;
mod message_handler;
pub use client::{Client, Disconnect, SendMessage, SendMessageBatch};

use crate::{
	feeds::{self, AnyFeed, ThreadFeed},
	message::Message,
	mt_context::MTAddr,
	registry::Registry,
	str_err,
	util::DynResult,
};
use actix::prelude::*;
use std::net::IpAddr;

/// Public key public and private ID set
#[derive(Clone, Default, Debug)]
struct PubKeyDesc {
	/// Public key private ID used to sign messages by the client
	priv_id: u64,

	/// Public key public ID used to sign messages by the client
	pub_id: uuid::Uuid,
}

/// Client connection state
#[derive(Debug)]
enum ConnState {
	/// Freshly established a WS connection
	Connected,

	/// Sent handshake message and it was accepted
	AcceptedHandshake,

	/// Public key already registered. Requested client to send a HandshakeReq
	/// with Authorization::Saved.
	RequestedReshake { pub_key: Vec<u8> },

	/// Client synchronized to a feed
	Synchronized { id: u64, feed: AnyFeed },
}

impl Default for ConnState {
	fn default() -> Self {
		Self::Connected
	}
}

#[derive(Debug)]
struct OpenPost {
	thread: u64,
	loc: feeds::PostLocation,
	body: Vec<char>,
	feed: MTAddr<ThreadFeed>,
}

/// Immutable client state set on client creation
#[derive(Debug)]
struct State {
	/// ID of client used in various registries
	id: u64,

	/// IP address of the client
	ip: IpAddr,

	/// Address to the global registry
	registry: Addr<Registry>,
}

/// Result of asynchronously processing a message with possible error
#[derive(Message)]
#[rtype(result = "()")]
struct WrappedMessageProcessingResult(pub DynResult<MessageProcessingResult>);

/// Result of asynchronously processing a message
struct MessageProcessingResult {
	/// Possibly modified mutable state
	mut_state: MutState,

	/// Possibly generated concatenated message
	message: Option<Message>,
}

/// Mutable client state affected by messages
#[derive(Debug, Default)]
struct MutState {
	/// Client connection state
	conn_state: ConnState,

	/// Post the client is currently editing
	open_post: Option<OpenPost>,

	/// Public key public and private ID set
	pub_key: PubKeyDesc,
}
