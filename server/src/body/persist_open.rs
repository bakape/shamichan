use crate::util::Pulse;
use actix::prelude::*;
use common::payloads::post_body::Node;
use std::{collections::HashMap, sync::Arc};

/// Periodically flushes open post bodies to the DB
#[derive(Default, Debug)]
pub struct BodyFlusher {
	bodies: HashMap<u64, Arc<Node>>,
	pending_pulse: Option<SpawnHandle>,
	flush_task: Option<SpawnHandle>,
}

impl Actor for BodyFlusher {
	type Context = Context<Self>;
}

impl BodyFlusher {
	/// Schedule processing of the buffered state in 1s, if not yet scheduled
	fn schedule_pulse(&mut self, ctx: &mut <Self as Actor>::Context) {
		if self.pending_pulse.is_none() {
			self.pending_pulse = ctx
				.notify_later(Pulse, std::time::Duration::from_secs(1))
				.into();
		}
	}
}

/// Asynchronously persist open post bodies to DB in batches
#[derive(Message)]
#[rtype(result = "()")]
pub struct PersistBodies(pub Vec<(u64, Arc<Node>)>);

impl Handler<PersistBodies> for BodyFlusher {
	type Result = ();

	fn handle(
		&mut self,
		bodies: PersistBodies,
		ctx: &mut Self::Context,
	) -> Self::Result {
		self.schedule_pulse(ctx);
		self.bodies.extend(bodies.0);
	}
}

impl Handler<Pulse> for BodyFlusher {
	type Result = ();

	fn handle(&mut self, _: Pulse, ctx: &mut Self::Context) -> Self::Result {
		self.pending_pulse = None;
		if self.flush_task.is_some() {
			self.schedule_pulse(ctx);
			return;
		}

		/// Runs flushing in a separate task with a passed snapshot to prevent
		/// lock contention on I/O
		async fn flush(
			addr: Addr<BodyFlusher>,
			bodies: HashMap<u64, Arc<Node>>,
		) {
			addr.send(FlushFinished(
				crate::db::write_open_post_bodies(bodies).await,
			));
		}

		self.flush_task = Some(
			ctx.spawn(
				flush(ctx.address(), std::mem::take(&mut self.bodies))
					.into_actor(self),
			),
		);
	}
}

/// Notify a flush has finished
#[derive(Message)]
#[rtype(result = "()")]
struct FlushFinished(crate::util::DynResult);

impl Handler<FlushFinished> for BodyFlusher {
	type Result = ();

	fn handle(
		&mut self,
		res: FlushFinished,
		_: &mut Self::Context,
	) -> Self::Result {
		self.flush_task = None;
		if let Err(e) = res.0 {
			log::error!("failed to flush open post bodies: {}", e);
		}
	}
}
