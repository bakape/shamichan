mod client;
mod message_handler;
pub use client::{Client, Disconnect, SendMessage, SendMessageBatch};

use crate::{
	feeds::IndexFeed, mt_context::MTAddr, registry::Registry, str_err,
};
use actix::Addr;
use std::net::IpAddr;

/// Immutable client state set on client creation
#[derive(Debug)]
struct State {
	/// ID of client used in various registries
	id: u64,

	/// IP address of the client
	ip: IpAddr,

	/// Address to the global registry
	registry: Addr<Registry>,

	/// Address of index feed
	index_feed: MTAddr<IndexFeed>,
}
