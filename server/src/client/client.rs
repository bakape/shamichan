use super::str_err;
use crate::{
	registry::{self, Registry},
	util::{self, DynResult},
};
use actix::prelude::*;
use actix_web_actors::ws;
use bytes::Bytes;
use std::{collections::VecDeque, net::IpAddr, rc::Rc};

/// Current state of handling or not handling messages by Client
#[derive(Debug)]
enum MessageHandling {
	/// Not currently handling a message
	NotHandling(super::MutState),

	/// Currently handling a message and can't handle another one till this
	/// one finishes
	Handling(SpawnHandle),
}

/// Maps to a websocket client on the Go side
//
// TODO: Split into mutable and immutable state and state passed back and
// forth between message handlers via enum. The enum is to be used to
// guarantee message handling exclusivity.
#[derive(Debug)]
pub struct Client {
	/// Immutable client state set on client creation
	state: Rc<super::State>,

	/// Buffered received messages
	received_buffer: VecDeque<Bytes>,

	/// Current state of handling or not handling messages by Client
	message_handling: MessageHandling,
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
					if self.received_buffer.len() >= 100 {
						str_err!("received message buffer exceeded");
					}
					self.received_buffer.push_back(buf);
					self.process_received(ctx);
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

impl Handler<super::WrappedMessageProcessingResult> for Client {
	type Result = ();

	fn handle(
		&mut self,
		msg: super::WrappedMessageProcessingResult,
		ctx: &mut Self::Context,
	) -> Self::Result {
		match msg.0 {
			Ok(res) => {
				if let Some(msg) = res.message {
					ctx.binary(msg);
				}
				self.message_handling =
					MessageHandling::NotHandling(res.mut_state);

				// Process next message, if any
				self.process_received(ctx);
			}
			Err(e) => self.fail(ctx, &e),
		};
	}
}

impl Client {
	/// Create fresh unconnected client
	pub fn new(ip: IpAddr, registry: Addr<Registry>) -> Self {
		lazy_static::lazy_static! {
			static ref ID_GEN: util::IDGenerator = Default::default();
		}

		Self {
			state: Rc::new(super::State {
				ip,
				registry,
				id: ID_GEN.next(),
			}),
			received_buffer: Default::default(),
			message_handling: MessageHandling::NotHandling(super::MutState {
				conn_state: super::ConnState::Connected,
				open_post: Default::default(),
				pub_key: Default::default(),
			}),
		}
	}

	/// Log critical client error and send it to the client and stop the Actor
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

	/// Process a buffered received message, if not already processing one
	fn process_received(&mut self, ctx: &mut <Self as Actor>::Context) {
		match &mut self.message_handling {
			MessageHandling::Handling(_) => (),
			MessageHandling::NotHandling(s) => {
				if let Some(msg) = self.received_buffer.pop_front() {
					self.message_handling = MessageHandling::Handling(
						ctx.spawn(
							super::message_handler::MessageHandler::new(
								self.state.clone(),
								std::mem::take(s),
							)
							.handle_message(ctx.address(), msg)
							.into_actor(self),
						),
					);
				}
			}
		}
	}
}
