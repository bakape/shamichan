use super::common::{Ctx, PostCommon, PostComponent};
use crate::{connection, state, util};
use common::MessageType;
use std::collections::HashMap;
use wasm_bindgen::JsCast;
use yew::{
	agent::{AgentLink, Bridge, Bridged, HandlerId},
	html,
	services::render::{RenderService, RenderTask},
	ComponentLink, Html, NodeRef,
};

/// A post actively being edited by the user
pub type PostForm = PostCommon<Inner>;

#[derive(Default)]
pub struct Inner {
	/// Must not be None after init() has been called
	#[allow(unused)]
	agent: Option<Box<dyn Bridge<Agent>>>,

	render_task: Option<RenderTask>,

	text_area: NodeRef,
	state: State,
}

#[derive(Clone)]
pub struct PostQuoteReq {
	id: u64,
	text: String,
}

#[derive(Clone)]
pub enum FormMessage {
	/// Set agent state
	SetState(State),

	/// Quote a post and include any selected text`
	QuotePost(PostQuoteReq),

	/// Textarea contents changed
	TextInput(String),

	/// Focus the textarea at position
	Focus {
		/// Position of cursor
		pos: u32,

		/// First of 2 sequential requests
		first: bool,
	},
}

impl PostComponent for Inner {
	type MessageExtra = FormMessage;

	fn init(&mut self, link: &ComponentLink<PostCommon<Self>>) {
		use super::common::Message::Extra;
		use Response::*;

		let mut a = Agent::bridge(link.callback(|msg| match msg {
			State(s) => Extra(FormMessage::SetState(s)),
			RenderQuoted(req) => Extra(FormMessage::QuotePost(req)),
		}));
		a.send(Request::SubViewUpdates);
		self.agent = a.into();
	}

	#[inline]
	fn should_render<'c>(&self, _: &Ctx<'c, Self>) -> bool {
		use State::*;

		match self.state {
			Draft { .. }
			| Allocated { .. }
			| Allocating { .. }
			| NeedCaptcha { .. }
			| CriticalError => true,
			Ready => false,
		}
	}

	#[inline]
	fn is_draggable<'c>(&self, _: &Ctx<'c, Self>) -> bool {
		true
	}

	fn update_extra<'c>(
		&mut self,
		c: &mut Ctx<'c, Self>,
		msg: Self::MessageExtra,
	) -> bool {
		util::with_logging(|| {
			use FormMessage::*;

			Ok(match msg {
				SetState(s) => {
					if self.state == s {
						return Ok(false);
					}

					match &s {
						State::Allocated { post } if post != &c.props().id => {
							let mut old = c.props().clone();
							old.id = *post;
							c.set_props(old);
						}
						_ => (),
					};

					// Focus input right after opening form
					if self.state == State::Ready {
						c.link().send_message(super::common::Message::Extra(
							Focus {
								pos: 0,
								first: true,
							},
						));
					}

					self.state = s;

					true
				}
				TextInput(s) => {
					self.resize_textarea()?;
					self.commit(s);
					false
				}
				QuotePost(req) => {
					use std::fmt::Write;

					let ta = self.text_area()?;
					let pos = ta.selection_end()?.unwrap_or(0);
					let old: Vec<char> = ta.value().chars().collect();
					let mut add_newline = false;
					let have_selection = !req.text.is_empty();

					// Insert post link and preceding whitespace.
					let mut s = match old.last() {
						// If empty post or newline before cursor, tell
						// next switch to do a newline after the quote.
						None | Some('\n') => {
							add_newline = true;
							format!(">>{}", req.id)
						}
						Some(' ') => format!(">>{}", req.id),
						_ => {
							if have_selection {
								format!("\n>>{}", req.id)
							} else {
								format!(" >>{}", req.id)
							}
						}
					};

					// Insert superseding whitespace after post link.
					match old.get(pos as usize) {
						None | Some(' ') | Some('\n') => {
							if add_newline || have_selection {
								s.push('\n');
							}
							add_newline = false;
						}
						_ => {
							add_newline = true;
							s.push(if have_selection { '\n' } else { ' ' });
						}
					};

					// If we do have a selection of text, then quote all lines.
					if have_selection {
						for line in req.text.lines() {
							write!(&mut s, ">{}\n", line)?;
						}
						if add_newline {
							s.push('\n');
						}
					}

					let s_chars: Vec<_> = s.chars().collect();
					let s_chars_len = s_chars.len();
					self.replace_text(
						&c.link(),
						{
							// Combine new body
							let old_len = old.len();
							let (old_l, old_r) = old.split_at(pos as usize);
							let mut new =
								String::with_capacity(old_len + s.len());
							new.extend(old_l);
							new.extend(s_chars);
							new.extend(old_r);
							new
						},
						{
							// Correct cursor position after inserting newline
							let mut new = pos as usize + s_chars_len;
							if add_newline {
								new -= 1;
							}
							new as u32
						},
						// Don't commit a quote, if it is the only input in a
						// post
						matches!(self.state, State::Draft { .. })
							|| old.is_empty(),
					)?;

					true
				}
				Focus { pos, first } => {
					let ta = self.text_area()?;
					ta.focus()?;
					ta.set_selection_range(pos, pos)?;

					// Because Firefox refocuses the clicked <a> on quote
					if first {
						self.on_next_frame(
							&c.link(),
							FormMessage::Focus { pos, first: false },
						);
					} else {
						self.render_task = None;
					}

					false
				}
			})
		})
	}

	#[inline]
	fn extra_classes<'c>(&self, _: &Ctx<'c, Self>) -> &'static [&'static str] {
		&["post-form"]
	}

	fn render_body<'c>(&self, c: &Ctx<'c, Self>) -> Html {
		html! {
			<textarea
				class="post-form-input"
				ref=self.text_area.clone()
				oninput=c.link().callback(|yew::html::InputData{value}| {
					super::common::Message::Extra(FormMessage::TextInput(value))
				})
			>
			</textarea>
		}
	}

	fn render_after<'c>(&self, _: &Ctx<'c, Self>) -> Html {
		html! {
			<span>{"TODO: controls"}</span>
		}
	}
}

impl Inner {
	/// Return input textarea element
	fn text_area(&self) -> util::Result<web_sys::HtmlTextAreaElement> {
		match self.text_area.get() {
			Some(el) => {
				el.dyn_into().map_err(|_| "not a textarea element".into())
			}
			None => Err("no textarea found".into()),
		}
	}

	/// Replace the current body and set the cursor to the input's end.
	/// commit: commit any changes to the server
	fn replace_text(
		&mut self,
		link: &ComponentLink<PostCommon<Self>>,
		body: String,
		pos: u32,
		commit: bool,
	) -> util::Result {
		self.text_area()?.set_value(&body);
		self.resize_textarea()?;
		if commit {
			self.commit(body);
		}
		self.on_next_frame(link, FormMessage::Focus { pos, first: true });
		Ok(())
	}

	/// Send message to execute on the next animation frame
	fn on_next_frame(
		&mut self,
		link: &ComponentLink<PostCommon<Self>>,
		msg: FormMessage,
	) {
		self.render_task = RenderService::request_animation_frame(
			link.callback(move |_| super::common::Message::Extra(msg.clone())),
		)
		.into();
	}

	/// Commit body changes to server
	fn commit(&mut self, body: String) {
		self.agent.as_mut().unwrap().send(Request::CommitText(body));
	}

	/// Resize textarea to content width and adjust height
	fn resize_textarea(&mut self) -> util::Result {
		let ta = self.text_area()?;
		let s = ta.style();

		macro_rules! set {
			($k:expr, $v:expr) => {
				s.set_property($k, &format!("{}px", $v))?;
			};
			($k:expr, $v:expr, $min:expr) => {
				let v = $v;
				set!($k, if v > $min { v } else { $min });
			};
		}

		set!("width", 0);
		set!("height", 0);
		ta.set_wrap("off");

		// Make the line slightly larger, so there is enough space for the next
		// character. This prevents wrapping on type.
		set!("width", ta.scroll_width(), 260);
		ta.set_wrap("soft");
		set!("height", ta.scroll_height(), 16);

		Ok(())
	}
}

/// State oif the agent FSM
#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum State {
	/// Ready to create posts. Post form closed.
	Ready,

	/// Post open but not yet allocating to target thread
	Draft { thread: u64 },

	/// Sent a request to allocate a post on target thread
	Allocating { thread: u64 },

	/// Post open and allocated to the server
	Allocated { post: u64 },

	/// Captcha solution required to allocate to target thread.
	/// This can only take place as an interrupt from the server during
	/// Allocating.
	#[allow(unused)] // TODO: remove it
	NeedCaptcha { thread: u64 },

	/// Client suffered unrecoverable error
	CriticalError,
}

impl Default for State {
	#[inline]
	fn default() -> Self {
		Self::Ready
	}
}

pub enum Message {
	/// Websocket connection state updated
	ConnStateUpdate(connection::State),

	/// User text selection has changed
	SelectionChange,

	/// A change in the currently open post ID
	OpenPostIDChange,
}

/// Currently selected text and elements
struct Selection {
	start: web_sys::Element,
	end: web_sys::Element,
	text: String,
}

pub enum Request {
	/// Quote a post and any selected text
	QuotePost {
		post: u64,
		target_post: web_sys::Node,
	},

	/// Register as a post form view
	SubViewUpdates,

	/// Commit text body changes
	CommitText(String),

	/// Open a draft postform for a target thread
	OpenDraft(u64),
}

enum Subscription {
	StateOnly,
	ViewUpdates,
}

#[derive(Clone)]
pub enum Response {
	/// Agent state update
	State(State),

	/// Render quoted post in view
	RenderQuoted(PostQuoteReq),
}

/// Only one PostForm can exist at a time so this agent manages it
pub struct Agent {
	state: State,
	link: AgentLink<Self>,
	subscribers: HashMap<HandlerId, Subscription>,

	#[allow(unused)]
	conn: Box<dyn Bridge<connection::Connection>>,
	app_state: state::StateBridge,

	conn_state: connection::State,

	/// Store last selected range, so we can access it after a mouse click on
	/// quote links, which cause that link to become selected
	last_selection: Option<Selection>,

	/// Current state of the open post body text
	post_body: Vec<char>,
}

impl yew::agent::Agent for Agent {
	type Reach = yew::agent::Context<Self>;
	type Message = Message;
	type Input = Request;
	type Output = Response;

	#[cold]
	fn create(link: AgentLink<Self>) -> Self {
		util::add_static_listener(
			util::document(),
			"selectionchange",
			true,
			link.callback(|_: web_sys::Event| Message::SelectionChange),
		);

		// TODO: locked thread handling
		// TODO: don't reset on thread change. Just keep floating.
		// TODO: claim existing open post on reconnection
		// TODO: use onbeforeunload to prevent accidental closing of postform
		// TODO: image insertion must specify target post
		Self {
			conn: connection::Connection::bridge(
				link.callback(|s| Message::ConnStateUpdate(s)),
			),
			app_state: state::hook(
				&link,
				vec![state::Change::OpenPostID],
				|| Message::OpenPostIDChange,
			),
			link,
			state: State::Ready,
			conn_state: connection::State::Loading,
			subscribers: Default::default(),
			last_selection: Default::default(),
			post_body: Default::default(),
		}
	}

	fn update(&mut self, msg: Self::Message) {
		use Message::*;

		match msg {
			ConnStateUpdate(cs) => {
				use connection::State as CS;
				use State as S;

				self.conn_state = cs;
				if cs == CS::CriticalError {
					self.set_state(S::CriticalError);
					return;
				}

				match self.state {
					// Nothing to transition into, if the form is just sitting
					// closed
					S::Ready => (),

					// If a full connection is established and we have something
					// in the draft, allocate a post
					S::Draft { .. } => match cs {
						CS::HandshakeComplete => self.commit_pending(),
						_ => (),
					},

					// If any connection change from HandshakeComplete occurred
					// during allocation, consider the post lost.
					S::Allocating { thread } | S::NeedCaptcha { thread } => {
						self.set_state(S::Draft { thread })
					}

					S::Allocated { post } => match cs {
						// TODO: resend body and try to resend any missing
						// buffered image, if a disconnect happened
						CS::HandshakeComplete => {}

						// Message filtering and deferral on connection loss
						// is handled by the connection module itself.
						_ => (),
					},

					// Already stalled on a critical error
					S::CriticalError => (),
				};
			}
			OpenPostIDChange => {
				match (&self.app_state.get().open_post_id, &self.state) {
					(Some(id), State::Ready { .. })
					| (Some(id), State::Draft { .. })
					| (Some(id), State::Allocating { .. }) => {
						self.set_state(State::Allocated { post: *id });
					}
					_ => (),
				};
			}
			SelectionChange => util::with_logging(|| {
				fn closest_el(
					n: Option<web_sys::Node>,
				) -> Option<web_sys::Element> {
					n.map(|n| {
						if n.node_type() == web_sys::Node::TEXT_NODE {
							n.parent_element()
						} else {
							n.dyn_into().ok()
						}
					})
					.flatten()
				}

				if let Some((sel, start, end)) = util::window()
					.get_selection()?
					.map(|sel| {
						match (
							closest_el(sel.anchor_node()),
							closest_el(sel.focus_node()),
						) {
							(Some(start), Some(end)) => {
								match start.parent_element() {
									Some(p)
										if !p
											.class_list()
											.contains("quote") =>
									{
										Some((sel, start, end))
									}
									_ => None,
								}
							}
							_ => None,
						}
					})
					.flatten()
				{
					self.last_selection = Selection {
						start,
						end,
						text: sel.to_string().into(),
					}
					.into()
				}
				Ok(())
			}),
		}
	}

	fn connected(&mut self, id: HandlerId) {
		self.subscribers.insert(id, Subscription::StateOnly);
		self.send_current_state(id);
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.subscribers.remove(&id);
	}

	fn handle_input(&mut self, req: Self::Input, h: HandlerId) {
		use Request::*;

		match req {
			QuotePost { post, target_post } => {
				self.quote_post(post, target_post);
			}
			SubViewUpdates => {
				self.subscribers.insert(h, Subscription::ViewUpdates);
			}
			CommitText(new) => self.commit_text(new.chars().collect()),
			OpenDraft(thread) => {
				if self.state == State::Ready {
					self.set_state(State::Draft { thread });
				}
			}
		}
	}
}

impl Agent {
	/// Set new state and send it to all subscribers
	fn set_state(&mut self, new: State) {
		if self.state != new {
			debug_log!(format!(
				"set postform state: {:?} -> {:?}",
				self.state, new
			));
			self.state = new;
			if matches!(new, State::Allocated { .. }) {
				self.commit_pending();
			}
			for id in self.subscribers.keys() {
				self.send_current_state(*id);
			}
		}
	}

	#[inline]
	fn send_current_state(&self, subscriber: HandlerId) {
		self.link.respond(subscriber, Response::State(self.state));
	}

	/// Send a message only to view subscribers
	fn send_to_views(&self, msg: &Response) {
		for h in self
			.subscribers
			.iter()
			.filter(|(_, s)| matches!(s, Subscription::ViewUpdates))
		{
			self.link.respond(*h.0, msg.clone())
		}
	}

	/// Try allocating a post, if it is eligible and not yet allocated.
	/// Returns, if a post is allocated or allocating.
	fn try_alloc(&mut self) -> bool {
		use State::*;

		match self.state {
			Draft { thread } => {
				let s = self.app_state.get();
				if !s.location.is_thread() {
					return false;
				}

				connection::send(
					MessageType::InsertPost,
					&common::payloads::PostCreationReq {
						thread,
						sage: s.new_post_opts.sage,
						opts: common::payloads::NewPostOpts {
							name: s.new_post_opts.name.clone(),
						},
					},
				);
				self.set_state(Allocating { thread });
				true
			}
			Allocating { .. } | Allocated { .. } => true,
			_ => false,
		}
	}

	fn quote_post(&mut self, post: u64, target_post: web_sys::Node) {
		util::with_logging(|| {
			if !self.try_alloc() {
				return Ok(());
			}

			let sel_text = match &self.last_selection {
				Some(sel) => {
					// Check, if selection bound is mid-post
					let in_middle =
						|el: &web_sys::Element| -> util::Result<bool> {
							Ok(el.closest("blockquote")?.is_some()
								&& el
									.closest("article")?
									.map(|el| {
										el.is_same_node(Some(&target_post))
									})
									.unwrap_or(false))
						};

					if in_middle(&sel.start)?
						|| sel.start.is_same_node(Some(&target_post))
							&& (in_middle(&sel.end)?
								|| match (
									sel.end.closest("article")?,
									target_post.next_sibling(),
								) {
									(Some(a), Some(b)) => {
										a.is_same_node(Some(&b))
									}
									_ => false,
								}) {
						sel.text.clone()
					} else {
						Default::default()
					}
				}
				_ => Default::default(),
			};

			self.send_to_views(&Response::RenderQuoted(PostQuoteReq {
				id: post,
				text: sel_text,
			}));

			Ok(())
		});
	}

	/// Diff and commit text changes to server
	fn commit_text(&mut self, new: Vec<char>) {
		if new.is_empty() && self.post_body.is_empty() {
			return;
		}

		if !self.try_alloc() {
			// Buffer post body till alloc
			self.post_body = new;
			return;
		}

		if new == self.post_body {
			return;
		}

		let len_diff = new.len() as isize - self.post_body.len() as isize;
		if len_diff == 1 && &new[..new.len() - 1] == self.post_body.as_slice() {
			// Commit a character appendage to the end of the line
			connection::send(MessageType::Append, &new[new.len() - 1]);
		} else if len_diff == -1
			&& &self.post_body[..self.post_body.len() - 1] == new.as_slice()
		{
			// Send a message removing the last character of the line
			connection::send(MessageType::Backspace, &());
		} else {
			// Commit any other text body change that is not an append or
			// backspace
			connection::send(
				MessageType::PatchPostBody,
				&common::payloads::post_body::TextPatch::new(
					&self.post_body,
					&new,
				),
			);
		}

		self.post_body = new;
	}

	/// Commit any pending text or images
	fn commit_pending(&mut self) {
		if !self.post_body.is_empty() {
			let b = std::mem::take(&mut self.post_body);
			self.commit_text(b);
		}

		// TODO: also commit any pending image
		// TODO: do nothing, if image already inserted and a new one is sent,
		// but allow cancelling an upload in progress
	}
}
