use actix::prelude::*;
use async_trait::async_trait;
use lazy_static::lazy_static;
use std::{
	collections::VecDeque,
	fmt::Debug,
	marker::PhantomData,
	sync::{Arc, RwLock},
	time::Duration,
};
use tokio::sync::Mutex as AsyncMutex;

// TODO: MTWeakAddr
// TODO: thorough unit tests

lazy_static! {
	/// Global multithreaded Tokio runtime
	pub static ref TOKIO_RUNTIME: tokio::runtime::Runtime =
		tokio::runtime::Builder::new_multi_thread()
		.enable_all()
		.build()
		.unwrap();
}

/// Context for executing actors in the global Tokio thread pool.
///
/// Actors executing using this context should not block on I/O or sending
/// messages to other actors. Excessive blocking on locks should also be
/// avoided. These actions block threads in the tokio thread pool, reducing its
/// throughput.
///
/// Actors using MTContext have unbounded mailboxes. All messages received
/// are queued and processed sequentially with exclusive access to the actor's
/// state.
#[derive(Debug)]
pub struct MTContext<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	/// Address of Scheduler managing this actor
	scheduler: Addr<Scheduler<A>>,

	/// Current actor state, usable from both the Scheduler and actor
	state: Arc<RwLock<ActorState>>,

	/// The actor stopped itself
	stopped_self: bool,
}

/// Run A in the Tokio thread pool with a MTContext
pub fn run<A>(actor: A) -> MTAddr<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	lazy_static! {
		/// Dedicated Arbiter for scheduling work to ensure better response
		/// times as these will be messaged from a lot of different places. The
		/// random Arbiters assigned automatically by actix are more likely to
		/// also be loaded with websocket message workloads.
		static ref SCHEDULER_ARBITER: Arbiter = Arbiter::new();
	}

	// Fighting the Actor API limitations
	MTAddr::new(Scheduler::start_in_arbiter(&SCHEDULER_ARBITER, move |_| {
		Scheduler::new(actor)
	}))
}

/// Able to handle a message asynchronously
#[async_trait]
pub trait AsyncHandler<M>
where
	Self: Actor<Context = MTContext<Self>> + Send + Debug,
	M: Send,
{
	/// Error type returned by `handle()`
	type Error: Debug;

	/// Executes the message on self
	async fn handle(
		&mut self,
		msg: M,
		ctx: &mut Self::Context,
	) -> Result<(), Self::Error>;

	/// Handle any errors originating from handle()
	fn handle_error(&mut self, _: &mut Self::Context, err: Self::Error) {
		use std::any::type_name;

		log::error!(
			"failed to handle message {} on actor {}: {:?}\nactor state: {:#?}",
			type_name::<M>(),
			type_name::<Self>(),
			err,
			self,
		);
	}
}

/// Address for sending messages to MTContext Actors
#[derive(Debug)]
pub struct MTAddr<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	scheduler: Addr<Scheduler<A>>,
}

impl<A> Clone for MTAddr<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	fn clone(&self) -> Self {
		Self {
			scheduler: self.scheduler.clone(),
		}
	}
}

impl<A> MTAddr<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	fn new(scheduler: Addr<Scheduler<A>>) -> Self {
		Self { scheduler }
	}

	/// Sends a message unconditionally, ignoring any potential errors
	pub fn do_send<M>(&self, msg: M)
	where
		M: Send + 'static,
		A: AsyncHandler<M>,
	{
		self.scheduler
			.do_send(QueueMessage::new(msg, Default::default()));
	}
}

impl<A> MTContext<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	/// Returns the address of the context
	#[allow(unused)]
	pub fn address(&self) -> MTAddr<A> {
		MTAddr::new(self.scheduler.clone())
	}

	/// Sends a message to self
	#[allow(unused)]
	pub fn notify<M>(&mut self, msg: M)
	where
		M: Send + 'static,
		A: AsyncHandler<M>,
	{
		self.notify_later(msg, Duration::default());
	}

	/// Sends a message to self after a specified period of time
	#[allow(unused)]
	pub fn notify_later<M>(&mut self, msg: M, after: Duration)
	where
		M: Send + 'static,
		A: AsyncHandler<M>,
	{
		self.scheduler.do_send(QueueMessage::new(msg, after));
	}

	/// Sends a message to self each interval
	#[allow(unused)]
	pub fn notify_interval<M>(&mut self, msg: M, interval: Duration)
	where
		M: Send + Clone + 'static,
		A: AsyncHandler<M>,
	{
		self.scheduler
			.do_send(QueueMessageInterval::new(msg, interval));
	}
}

/// Message wrapped for handling either once or many times
#[async_trait]
trait WrappedMessage<A>: Debug + Send
where
	A: Actor<Context = MTContext<A>> + Send,
{
	/// Executes the contained message on the actor
	async fn handle(&mut self, act: &mut A, ctx: &mut A::Context);
}

/// Clonable message wrapped for handling many times
#[async_trait]
trait ClonableWrappedMessage<A>: WrappedMessage<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	fn as_wrapped_message(&self) -> Box<dyn WrappedMessage<A>>;
}

/// Envelope wrapping a message that can be handled only once
struct OneShotEnvelope<A, M>
where
	A: Actor<Context = MTContext<A>> + Send + AsyncHandler<M>,
	M: Send,
{
	msg: Option<M>,
	pd: PhantomData<A>,
}

impl<A, M> Debug for OneShotEnvelope<A, M>
where
	A: Actor<Context = MTContext<A>> + Send + AsyncHandler<M>,
	M: Send,
{
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		f.write_str("OneShotEnvelope(...)")
	}
}

#[async_trait]
impl<A, M> WrappedMessage<A> for OneShotEnvelope<A, M>
where
	A: Actor<Context = MTContext<A>> + Send + AsyncHandler<M>,
	M: Send,
{
	async fn handle(&mut self, act: &mut A, ctx: &mut A::Context) {
		if let Some(msg) = self.msg.take() {
			if let Err(e) = act.handle(msg, ctx).await {
				act.handle_error(ctx, e);
			}
		}
	}
}

/// Envelope wrapping a message that can be handled multiple times
struct ClonableEnvelope<A, M>
where
	A: Actor<Context = MTContext<A>> + Send + AsyncHandler<M>,
	M: Send + Clone,
{
	msg: M,
	pd: PhantomData<A>,
}

impl<A, M> Clone for ClonableEnvelope<A, M>
where
	A: Actor<Context = MTContext<A>> + Send + AsyncHandler<M>,
	M: Send + Clone,
{
	fn clone(&self) -> Self {
		Self {
			msg: self.msg.clone(),
			pd: PhantomData,
		}
	}
}

#[async_trait]
impl<A, M> WrappedMessage<A> for ClonableEnvelope<A, M>
where
	A: Actor<Context = MTContext<A>> + Send + AsyncHandler<M>,
	M: Send + Clone,
{
	async fn handle(&mut self, act: &mut A, ctx: &mut A::Context) {
		let msg = self.msg.clone();
		if let Err(e) = act.handle(msg, ctx).await {
			act.handle_error(ctx, e);
		}
	}
}

impl<A, M> ClonableWrappedMessage<A> for ClonableEnvelope<A, M>
where
	A: Actor<Context = MTContext<A>> + Send + AsyncHandler<M>,
	M: Send + Clone + 'static,
{
	fn as_wrapped_message(&self) -> Box<dyn WrappedMessage<A>> {
		Box::new(self.clone()) as Box<dyn WrappedMessage<A>>
	}
}

impl<A, M> Debug for ClonableEnvelope<A, M>
where
	A: Actor<Context = MTContext<A>> + Send + AsyncHandler<M>,
	M: Send + Clone,
{
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		f.write_str("ClonableEnvelope(...)")
	}
}

/// Set a stopping state on the context, if applicable.
/// Prevents concurrently overwriting higher priority states.
#[inline]
fn set_stopping_state(current: &Arc<RwLock<ActorState>>, new: ActorState) {
	use ActorState::*;

	let mut s = current.write().unwrap();
	match (new, *s) {
		(_, Running) | (Stopped, _) => {
			*s = new;
		}
		_ => (),
	}
}

impl<A> ActorContext for MTContext<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	fn stop(&mut self) {
		self.stopped_self = true;
		set_stopping_state(&self.state, ActorState::Stopping);
	}

	fn terminate(&mut self) {
		self.stopped_self = true;
		set_stopping_state(&self.state, ActorState::Stopped);
	}

	fn state(&self) -> ActorState {
		*self.state.read().unwrap()
	}
}

/// Schedules message execution between the Actix and Tokio runtimes
#[derive(Debug)]
struct Scheduler<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	/// Managed actor
	actor: Arc<AsyncMutex<A>>,

	/// Managed actor state
	state: Arc<RwLock<ActorState>>,

	/// The actor stopped itself
	stopped_self: bool,

	/// Messages pending handling
	pending: VecDeque<Box<dyn WrappedMessage<A>>>,
}

impl<A> Scheduler<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	fn new(actor: A) -> Self {
		Self {
			state: Arc::new(RwLock::new(ActorState::Started)),
			actor: Arc::new(AsyncMutex::new(actor)),
			pending: Default::default(),
			stopped_self: false,
		}
	}

	// Create a new MTContext instance
	fn ctx(&self, ctx: &mut <Self as Actor>::Context) -> MTContext<A> {
		MTContext::<A> {
			scheduler: ctx.address(),
			state: self.state.clone(),
			stopped_self: false,
		}
	}

	/// Run any pending actions in the queue, if none are running already
	fn try_run_pending(&mut self, ctx: &mut <Self as Actor>::Context) {
		if self.pending.is_empty() {
			return;
		}
		if let Ok(mut act) = self.actor.clone().try_lock_owned() {
			let scheduler = ctx.address();
			let pending = std::mem::take(&mut self.pending);
			let mut ctx = self.ctx(ctx);
			TOKIO_RUNTIME.spawn(async move {
				// Process all available messages in one batch to avoid
				// switching overhead
				let act: &mut A = &mut act; // cache dereference
				let mut it = pending.into_iter();
				while let Some(mut msg) = it.next() {
					msg.handle(act, &mut ctx).await;

					// Handle stopping back in the Scheduler or with async
					// termination routine
					if ctx.stopped_self
						|| matches!(
							ctx.state(),
							ActorState::Stopped | ActorState::Stopping
						) {
						break;
					}
				}
				scheduler.do_send(Done {
					unprocessed: it.collect(),
					stopped_self: ctx.stopped_self,
				});
			});
		}
	}

	/// Queue a message to be handled by A
	fn queue_message(
		&mut self,
		ctx: &mut <Self as Actor>::Context,
		msg: Box<dyn WrappedMessage<A>>,
	) {
		self.pending.push_back(msg);
		self.try_run_pending(ctx);
	}
}

impl<A: Actor> Actor for Scheduler<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	type Context = Context<Self>;

	fn started(&mut self, ctx: &mut Self::Context) {
		*self.state.write().unwrap() = ActorState::Running;
		// No events processing yet, so will never fail to lock
		self.actor.try_lock().unwrap().started(&mut self.ctx(ctx));

		// Remove practically any mailbox limits to make messages undroppable
		ctx.set_mailbox_capacity(1 << 20);
	}

	fn stopping(&mut self, ctx: &mut Self::Context) -> Running {
		// Don't stop, if there are still pending messages and the actor did not
		// stop itself
		if !self.pending.is_empty() && !self.stopped_self {
			return Running::Continue;
		}

		match self.actor.try_lock() {
			Ok(mut act) => {
				// Consume request to stop self, if any
				self.stopped_self = false;

				let s = act.stopping(&mut self.ctx(ctx));
				*self.state.write().unwrap() = match s {
					Running::Continue => ActorState::Running,
					Running::Stop => ActorState::Stopped,
				};
				s
			}
			// Defer stopping till after the actor returns from message
			// handling
			Err(_) => {
				set_stopping_state(&self.state, ActorState::Stopping);
				Running::Continue
			}
		}
	}

	fn stopped(&mut self, ctx: &mut Self::Context) {
		*self.state.write().unwrap() = ActorState::Stopped;

		let mut ctx = self.ctx(ctx);
		match self.actor.clone().try_lock_owned() {
			Ok(mut act) => act.stopped(&mut ctx),
			// Run the actor's stopped() after it completes its work.
			// This is needed because Scheduler will not receive Done from the
			// actor in the Tokio runtime anymore to call its stopped() method.
			Err(_) => {
				let act = self.actor.clone();
				TOKIO_RUNTIME.spawn(async move {
					act.lock().await.stopped(&mut ctx);
				});
			}
		}
	}
}

/// Queue a message for processing
#[derive(Message)]
#[rtype(result = "()")]
struct QueueMessage<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	/// Message to process
	msg: Box<dyn WrappedMessage<A>>,

	/// Delay to queue the message after. If 0, queues immediately.
	after: Duration,
}

impl<A> QueueMessage<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	fn new<M>(msg: M, after: Duration) -> Self
	where
		A: AsyncHandler<M>,
		M: Send + 'static,
	{
		Self {
			msg: Box::new(OneShotEnvelope::<A, M> {
				msg: msg.into(),
				pd: PhantomData,
			}),
			after,
		}
	}
}

impl<A> Handler<QueueMessage<A>> for Scheduler<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	type Result = ();

	fn handle(&mut self, mut msg: QueueMessage<A>, ctx: &mut Self::Context) {
		if msg.after == Duration::default() {
			self.queue_message(ctx, msg.msg);
		} else {
			let dur = std::mem::take(&mut msg.after);
			ctx.notify_later(msg, dur);
		}
	}
}

/// Queue a message for processing and resend it each interval
#[derive(Message)]
#[rtype(result = "()")]
struct QueueMessageInterval<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	/// Message to process
	msg: Box<dyn ClonableWrappedMessage<A>>,

	/// Interval to send message at
	interval: Duration,
}

impl<A> QueueMessageInterval<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	fn new<M>(msg: M, interval: Duration) -> Self
	where
		A: AsyncHandler<M>,
		M: Send + Clone + 'static,
	{
		Self {
			msg: Box::new(ClonableEnvelope::<A, M> {
				msg,
				pd: PhantomData,
			}),
			interval,
		}
	}
}

impl<A> Handler<QueueMessageInterval<A>> for Scheduler<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	type Result = ();

	fn handle(
		&mut self,
		QueueMessageInterval { msg, interval }: QueueMessageInterval<A>,
		ctx: &mut Self::Context,
	) {
		ctx.run_interval(interval, move |this, ctx| {
			this.queue_message(ctx, msg.as_wrapped_message());
		});
	}
}

/// Notify a message is done handling and send the actor back to the Scheduler.
/// Sends any not yet processed messages to be put at the start of the queue.
#[derive(Message)]
#[rtype(result = "()")]
struct Done<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	/// Messages not processed in this run due to stopping the actor
	unprocessed: Vec<Box<dyn WrappedMessage<A>>>,

	/// The actor stopped itself
	stopped_self: bool,
}

impl<A> Handler<Done<A>> for Scheduler<A>
where
	A: Actor<Context = MTContext<A>> + Send,
{
	type Result = ();

	fn handle(
		&mut self,
		Done {
			unprocessed,
			stopped_self,
		}: Done<A>,
		ctx: &mut Self::Context,
	) -> Self::Result {
		for msg in unprocessed.into_iter().rev() {
			self.pending.push_front(msg);
		}
		match *self.state.clone().read().unwrap() {
			ActorState::Stopped => {
				ctx.terminate();
			}
			ActorState::Stopping => {
				// Run the actor's stopping() on the next stopping() call of the
				// Scheduler
				self.stopped_self = stopped_self;
				ctx.stop();

				//  Schedule message handling, if actor is not stopped
				ctx.notify(Done {
					unprocessed: vec![],
					stopped_self: false,
				});
			}
			ActorState::Running => {
				self.try_run_pending(ctx);
			}
			_ => (),
		};
	}
}
