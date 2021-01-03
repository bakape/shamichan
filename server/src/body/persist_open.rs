use crate::{
	mt_context::{AsyncHandler, MTContext},
	util::{self, Pulse},
};
use actix::prelude::*;
use async_trait::async_trait;
use common::payloads::post_body::Node;
use std::{collections::HashMap, sync::Arc};

/// Periodically flushes open post bodies to the DB
#[derive(Default, Debug)]
pub struct BodyFlusher {
	bodies: HashMap<u64, Arc<Node>>,
	pending_pulse: bool,
}

impl Actor for BodyFlusher {
	type Context = MTContext<Self>;
}

impl BodyFlusher {
	/// Schedule processing of the buffered state in 1s, if not yet scheduled
	fn schedule_pulse(&mut self, ctx: &mut <Self as Actor>::Context) {
		if !self.pending_pulse {
			self.pending_pulse = true;
			ctx.notify_later(Pulse, std::time::Duration::from_secs(1));
		}
	}
}

/// Asynchronously persist open post bodies to DB in batches
pub struct PersistBodies(pub Vec<(u64, Arc<Node>)>);

#[async_trait]
impl AsyncHandler<PersistBodies> for BodyFlusher {
	type Error = util::Err;

	async fn handle(
		&mut self,
		PersistBodies(bodies): PersistBodies,
		ctx: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		self.schedule_pulse(ctx);
		self.bodies.extend(bodies);
		Ok(())
	}
}

#[async_trait]
impl AsyncHandler<Pulse> for BodyFlusher {
	type Error = util::Err;

	async fn handle(
		&mut self,
		_: Pulse,
		_: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		self.pending_pulse = false;
		crate::db::write_open_post_bodies(std::mem::take(&mut self.bodies))
			.await?;
		Ok(())
	}
}
