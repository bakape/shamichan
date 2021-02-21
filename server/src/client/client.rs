use super::{
	message_handler::{HandleMessage, MessageHandler, MessageResult},
	str_err,
};
use crate::{
	feeds::IndexFeed,
	message::Message as Msg,
	mt_context::MTAddr,
	registry::{self, Registry},
	util::{self, DynResult},
};
use actix::prelude::*;
use actix_web_actors::ws;
use std::{net::IpAddr, sync::Arc};

/// Client instance controller
#[derive(Debug)]
pub struct Client {
	/// Immutable client state set on client creation
	state: Arc<super::State>,

	/// Actor handling messages on the tokio multithreaded runtime
	message_handler: Option<MTAddr<MessageHandler>>,
}

impl Actor for Client {
	type Context = ws::WebsocketContext<Self>;

	fn started(&mut self, ctx: &mut Self::Context) {
		let ref s: super::State = *self.state;
		s.registry
			.send(registry::RegisterClient {
				id: s.id,
				addr: ctx.address(),
			})
			.into_actor(self)
			.then(|res, this, ctx| {
				if let Err(e) = res {
					this.fail(ctx, &e.into());
				}
				fut::ready(())
			})
			.wait(ctx);
	}

	#[cold]
	fn stopped(&mut self, _: &mut Self::Context) {
		let ref s: super::State = *self.state;
		s.registry.do_send(registry::UnregisterClient(s.id));
	}
}

impl StreamHandler<Result<ws::Message, ws::ProtocolError>> for Client {
	fn handle(
		&mut self,
		msg: Result<ws::Message, ws::ProtocolError>,
		ctx: &mut Self::Context,
	) {
		use ws::Message::*;

		if let Err(err) = (|| -> DynResult {
			match msg? {
				Binary(buf) => {
					match &mut self.message_handler {
						Some(h) => h,
						None => {
							self.message_handler = Some(
								crate::mt_context::run(MessageHandler::new(
									self.state.clone(),
									ctx.address(),
								)),
							);
							match &mut self.message_handler {
								Some(h) => h,
								None => unsafe {
									std::hint::unreachable_unchecked()
								},
							}
						}
					}
					.do_send(HandleMessage(buf));
				}
				Text(_) => str_err!("non-binary message received"),
				Continuation(_) => {
					str_err!("continuation messages not supported")
				}
				Close(_) => {
					ctx.stop();
				}
				Ping(_) | Pong(_) => {
					// TODO: ping once a minute and handle pongs
					// TODO: ping and pong support
					// TODO: prevent ping spam
					// TODO: reply to multiple pings with only one pong
				}
				Nop => (),
			};
			Ok(())
		})() {
			self.fail(ctx, &err);
		}
	}
}

impl Handler<MessageResult> for Client {
	type Result = ();

	fn handle(
		&mut self,
		msg: MessageResult,
		ctx: &mut Self::Context,
	) -> Self::Result {
		match msg.0 {
			Ok(Some(msg)) => ctx.binary(msg),
			Ok(None) => (),
			Err(e) => self.fail(ctx, &e),
		};
	}
}

/// Disconnect client with provided error
#[derive(Message)]
#[rtype(result = "()")]
pub struct Disconnect(pub util::Err);

impl Handler<Disconnect> for Client {
	type Result = ();

	fn handle(
		&mut self,
		msg: Disconnect,
		ctx: &mut Self::Context,
	) -> Self::Result {
		self.fail(ctx, &msg.0);
	}
}

/// Send message to client
#[derive(Message, Clone)]
#[rtype(result = "()")]
pub struct SendMessage(pub Msg);

impl Handler<SendMessage> for Client {
	type Result = ();

	fn handle(
		&mut self,
		msg: SendMessage,
		ctx: &mut Self::Context,
	) -> Self::Result {
		ctx.binary(msg.0);
	}
}

/// Send a batch of messages to client.
///
/// Wrapped in an Arc to reduce RC load on the individual messages.
#[derive(Message, Clone)]
#[rtype(result = "()")]
pub struct SendMessageBatch(Arc<Vec<Msg>>);

impl SendMessageBatch {
	#[inline]
	pub fn new(messages: Vec<Msg>) -> Self {
		Self(messages.into())
	}
}

impl Handler<SendMessageBatch> for Client {
	type Result = ();

	fn handle(
		&mut self,
		msg: SendMessageBatch,
		ctx: &mut Self::Context,
	) -> Self::Result {
		for m in msg.0.iter() {
			ctx.binary(m.clone());
		}
	}
}

impl Client {
	/// Create fresh unconnected client
	pub fn new(
		ip: IpAddr,
		registry: Addr<Registry>,
		index_feed: MTAddr<IndexFeed>,
	) -> Self {
		lazy_static::lazy_static! {
			static ref ID_GEN: util::IDGenerator = Default::default();
		}

		Self {
			state: Arc::new(super::State {
				ip,
				registry,
				index_feed,
				id: ID_GEN.next(),
			}),
			message_handler: None,
		}
	}

	/// Log critical client error and send it to the client and stop the Actor
	#[cold]
	fn fail(&self, ctx: &mut <Self as Actor>::Context, err: &util::Err) {
		// TODO: filter errors somehow (probably using error classes and an
		// internal error type instead of just util::Error)
		log::error!("websockets error by {}: {}", self.state.ip, err);

		ctx.close(Some(ws::CloseReason {
			code: ws::CloseCode::Protocol,
			description: Some(format!("error: {}", err)),
		}));
		// TODO: does stopping right after issuing a close send the close
		// message?
		ctx.stop();
	}
}
