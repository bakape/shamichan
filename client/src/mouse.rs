use common::debug_log;
use yew::{
	agent::{AgentLink, HandlerId},
	services::render::{RenderService, RenderTask},
};

#[derive(Default, Copy, Clone, Debug)]
pub struct Coordinates {
	pub x: i32,
	pub y: i32,
}

impl Coordinates {
	#[inline]
	pub fn is_zero(&self) -> bool {
		self.x == 0 && self.y == 0
	}
}

impl std::ops::Add for Coordinates {
	type Output = Self;

	#[inline]
	fn add(mut self, rhs: Self) -> Self {
		self.x += rhs.x;
		self.y += rhs.y;
		self
	}
}

impl std::ops::Sub for Coordinates {
	type Output = Self;

	#[inline]
	fn sub(mut self, rhs: Self) -> Self {
		self.x -= rhs.x;
		self.y -= rhs.y;
		self
	}
}

impl std::ops::AddAssign for Coordinates {
	#[inline]
	fn add_assign(&mut self, rhs: Self) {
		*self = *self + rhs;
	}
}

impl From<&web_sys::MouseEvent> for Coordinates {
	#[inline]
	fn from(e: &web_sys::MouseEvent) -> Self {
		Self {
			x: e.client_x(),
			y: e.client_y(),
		}
	}
}

/// Global mouse tracking agent. Sends new mouse Coordinates on mousemove.
pub struct Agent {
	link: AgentLink<Self>,
	current: Coordinates,
	render_task: Option<RenderTask>,

	state_subs: std::collections::HashSet<HandlerId>,

	// Hashmap instead of Option to prevent possible overlap in subs/unsub
	dragging: std::collections::HashSet<HandlerId>,
}

pub enum Message {
	MouseMove(Coordinates),
	MouseUp,
	AnimationFrame,
}

pub enum Request {
	StartDragging,
}

#[derive(Clone)]
pub enum Response {
	/// Is any element being dragged?
	IsDragging(bool),

	/// Mouse coordinates to drag element to
	Coordinates(Coordinates),

	/// Signals a dragging view it no longer is being dragged
	StoppedDragging,
}

impl yew::agent::Agent for Agent {
	type Reach = yew::agent::Context<Self>;
	type Message = Message;
	type Input = Request;
	type Output = Response;

	#[cold]
	fn create(link: AgentLink<Self>) -> Self {
		let doc = crate::util::document();
		crate::util::add_static_listener(
			doc,
			"mousemove",
			true,
			link.callback(|e: web_sys::MouseEvent| {
				Message::MouseMove(Coordinates::from(&e))
			}),
		);
		crate::util::add_static_listener(
			doc,
			"mouseup",
			true,
			link.callback(|_: web_sys::MouseEvent| Message::MouseUp),
		);
		Self {
			link,
			current: Default::default(),
			render_task: Default::default(),
			state_subs: Default::default(),
			dragging: Default::default(),
		}
	}

	fn connected(&mut self, id: HandlerId) {
		self.state_subs.insert(id);
	}

	fn handle_input(&mut self, req: Self::Input, id: HandlerId) {
		match req {
			Request::StartDragging => {
				debug_log!("started dragging", id);

				let was_empty = self.dragging.is_empty();
				self.dragging.insert(id);
				self.link.respond(id, Response::Coordinates(self.current));
				if was_empty {
					self.send_dragging_status();
				}
			}
		}
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.state_subs.remove(&id);
		if self.dragging.remove(&id) && self.dragging.is_empty() {
			debug_log!("stopped dragging", id);
			self.send_dragging_status();
		}
	}

	fn update(&mut self, msg: Self::Message) {
		match msg {
			Message::MouseMove(coords) => {
				self.current = coords;

				if self.dragging.is_empty() {
					return;
				}

				if self.render_task.is_none() {
					self.render_task = RenderService::request_animation_frame(
						self.link.callback(|_| Message::AnimationFrame),
					)
					.into();
				}
			}
			Message::AnimationFrame => {
				self.render_task = None;
				let msg = Response::Coordinates(self.current);
				for sub in self.dragging.iter().copied() {
					self.link.respond(sub, msg.clone());
				}
			}
			Message::MouseUp => {
				let dragging = std::mem::take(&mut self.dragging);
				if !dragging.is_empty() {
					for id in dragging {
						self.link.respond(id, Response::StoppedDragging);
					}
					debug_log!("stopped dragging", "all");
					self.send_dragging_status();
				}
			}
		}
	}
}

impl Agent {
	fn send_dragging_status(&self) {
		let msg = Response::IsDragging(!self.dragging.is_empty());
		for id in self.state_subs.iter() {
			self.link.respond(*id, msg.clone());
		}
	}
}
