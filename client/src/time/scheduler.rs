/// Centralized agent for efficiently scheduling timer updates
use super::queue::Queue;
use crate::state;
use yew::agent::{Agent, AgentLink, Context, HandlerId};
use yew::services::interval::{IntervalService, IntervalTask};

// TODO: apply correction from server clock

/// Unit division/multiplication array for computing a time unit from seconds
static UNITS: [(Unit, u8); 5] = [
	(Unit::Seconds, 60),
	(Unit::Minutes, 60),
	(Unit::Hours, 24),
	(Unit::Days, 30),
	(Unit::Months, 12),
];

/// Agent that updates all Time components, if their value changed within
/// their current resolution or options changed
pub struct Scheduler {
	link: AgentLink<Self>,
	use_relative: bool,
	now: u32,
	time_correction: i32,

	// Prevent dropping
	#[allow(unused)]
	app_state: state::StateBridge,
	#[allow(unused)]
	interval: IntervalTask,

	queue: Queue<Tick>,
}

pub enum Message {
	Tick,
	StateUpdated,
}

#[derive(Copy, Clone, Eq, PartialEq)]
pub enum Unit {
	Seconds,
	Minutes,
	Hours,
	Days,
	Months,
	Years,
}

impl Default for Unit {
	#[inline]
	fn default() -> Self {
		Self::Seconds
	}
}

#[derive(Copy, Clone, Default)]
pub struct RelativeTime {
	pub is_future: bool,
	pub duration: u8,
	pub unit: Unit,
}

impl std::fmt::Display for RelativeTime {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		if self.duration == 0 {
			write!(f, "{}", localize!("now"))?;
			return Ok(());
		}

		write!(
			f,
			"{}",
			localize!(
				if self.is_future {
					"time_in"
				} else {
					"time_ago"
				},
				{
					"number" => &self.duration.to_string()
					"unit" => crate::lang::pluralize(
						match self.unit {
							Unit::Seconds => "second",
							Unit::Minutes => "minute",
							Unit::Hours => "hour",
							Unit::Days => "day",
							Unit::Months => "month",
							Unit::Years => "year",
						},
						self.duration,
					)
				}
			)
		)
	}
}

impl RelativeTime {
	/// Compute current relative timestamp
	fn new(now: u32, val: u32) -> Self {
		let mut time = now as i64 - val as i64;
		let mut is_future = false;
		if time < 0 {
			time = -time;
			is_future = true;
		}

		#[rustfmt::skip]
		macro_rules! pack {
			($unit:expr) => {
				return RelativeTime {
					is_future: is_future,
					duration: time as u8,
					unit: $unit,
				}
			};
		}

		for u in UNITS.iter() {
			if time < u.1 as i64 {
				pack!(u.0)
			}
			time /= u.1 as i64;
		}

		pack!(Unit::Years)
	}
}

/// A clock update pending at a known time
struct Tick {
	/// Subscriber ID
	id: HandlerId,
	/// Value of subscriber
	val: u32,
	/// Time of next update
	pending_on: u32,
	/// Value of next update
	diff: RelativeTime,
}

impl PartialEq for Tick {
	#[inline]
	fn eq(&self, other: &Tick) -> bool {
		self.pending_on == other.pending_on
	}
}

impl PartialOrd for Tick {
	#[inline]
	fn partial_cmp(&self, other: &Tick) -> Option<std::cmp::Ordering> {
		self.pending_on.partial_cmp(&other.pending_on)
	}
}

impl Tick {
	/// Create a new tick at the current moment in time
	#[inline]
	fn new(id: HandlerId, val: u32, now: u32) -> Self {
		Self {
			id,
			val,
			pending_on: now,
			diff: RelativeTime::new(now, val),
		}
	}

	/// Compute relative timestamp at next tick and next tick pending time
	fn set_next_tick(&mut self, mut now: u32) {
		// Floor the current timestamp to the minimum point of the current tick
		// and then add the full tick time
		let mut divisor: u32 = 1;
		for (unit, max) in UNITS.iter() {
			if self.diff.unit == *unit {
				break;
			}
			divisor *= *max as u32;
		}
		now = now - (now % divisor) + divisor;

		self.diff = RelativeTime::new(now, self.val);
		self.pending_on = now;
	}
}

/// The current time + bool does not actually take more memory as an enum, so
/// might as well send both on each Scheduler
#[derive(Clone, Default)]
pub struct Response {
	pub diff: RelativeTime,
	pub use_relative: bool,
}

#[derive(Clone)]
pub enum Request {
	/// Register a new timer with passed Unix timestamp
	Register(u32),

	/// Change unix timestamp on an exiting timer
	ChangeTimestamp(u32),
}

impl Agent for Scheduler {
	type Reach = Context<Self>;
	type Message = Message;
	type Input = Request;
	type Output = Response;

	#[cold]
	fn create(link: AgentLink<Self>) -> Self {
		use state::Change;

		let mut this = Self {
			interval: IntervalService::spawn(
				std::time::Duration::from_secs(1),
				link.callback(|_| Message::Tick),
			),
			app_state: state::hook(
				&link,
				vec![Change::Options, Change::TimeCorrection],
				|| Message::StateUpdated,
			),
			link,
			use_relative: false,
			now: 0,
			time_correction: 0,
			queue: Default::default(),
		};
		let s = this.app_state.get();
		this.use_relative = s.options.relative_timestamps;
		this.time_correction = s.time_correction;
		this.update_now();
		this
	}

	fn update(&mut self, msg: Self::Message) {
		match msg {
			Message::StateUpdated => {
				let mut resend = false;
				let s = self.app_state.get();

				if s.options.relative_timestamps != self.use_relative {
					self.use_relative = s.options.relative_timestamps;
					resend = true;
				}

				if self.time_correction != s.time_correction {
					self.time_correction = s.time_correction;
					self.update_now();
					resend = true;
					for t in self.queue.iter_mut() {
						t.diff = RelativeTime::new(self.now, t.val);
					}
				}

				if resend {
					for t in self.queue.iter() {
						self.send(&t);
					}
				}
			}
			Message::Tick => {
				self.update_now();
				loop {
					let pop = match self.queue.peek() {
						Some(peeking) => {
							if peeking.pending_on > self.now {
								break;
							}
							true
						}
						None => break,
					};
					if pop {
						let t = self.queue.pop().unwrap();
						self.refresh_tick(t);
					}
				}
			}
		}
	}

	fn handle_input(&mut self, req: Self::Input, id: HandlerId) {
		use Request::*;

		match req {
			Register(time) => self.refresh_tick(Tick::new(id, time, self.now)),
			ChangeTimestamp(time) => {
				self.remove(id);
				self.refresh_tick(Tick::new(id, time, self.now));
			}
		}
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.remove(id);
	}
}

impl Scheduler {
	/// Send a tick to a subscriber
	fn send(&self, t: &Tick) {
		self.link.respond(
			t.id,
			Response {
				diff: t.diff,
				use_relative: self.use_relative,
			},
		);
	}

	/// Send the current Tick and queue the next Tick
	fn refresh_tick(&mut self, mut t: Tick) {
		self.send(&t);
		t.set_next_tick(self.now);
		self.queue.insert(t);
	}

	/// Remove a registered listener
	fn remove(&mut self, id: HandlerId) {
		self.queue.remove(&HandlerIDKey(&id));
	}

	/// Update current Unix timestamp corrected for drift between server and client
	#[inline]
	fn update_now(&mut self) {
		self.now =
			(crate::util::now() as i32 + self.time_correction as i32) as u32;
	}
}

/// Enables eviction of queued Ticks by HandlerId
struct HandlerIDKey<'a>(&'a HandlerId);

impl<'a> PartialEq<Tick> for HandlerIDKey<'a> {
	#[inline]
	fn eq(&self, other: &Tick) -> bool {
		self.0 == &other.id
	}
}
