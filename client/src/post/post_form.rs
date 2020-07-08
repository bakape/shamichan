use super::common::{PostCommon, PostComponent, RenderCtx};
use crate::{connection, state, util};
use protocol::MessageType;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use wasm_bindgen::JsCast;
use yew::{
	agent::{AgentLink, Bridge, Bridged, HandlerId},
	html,
	services::render::{RenderService, RenderTask},
	ComponentLink, Html, NodeRef,
};

// A post actively being edited by the user
pub type PostForm = PostCommon<Inner>;

#[derive(Default)]
pub struct Inner {
	// Must not be None after init() has been called
	#[allow(unused)]
	agent: Option<Box<dyn Bridge<Agent>>>,

	render_task: Option<RenderTask>,

	text_area: NodeRef,
	state: State,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct PostQuoteReq {
	id: u64,
	text: String,
}

#[derive(Clone)]
pub enum FormMessage {
	// Set agent state
	SetState(State),

	// Quote a post and include any selected text`
	QuotePost(PostQuoteReq),

	// Focus the textarea at position
	Focus {
		// Position of cursor
		pos: u32,

		// First of 2 sequential requests
		first: bool,
	},
}

impl PostComponent for Inner {
	type MessageExtra = FormMessage;

	fn init(&mut self, link: &ComponentLink<PostCommon<Self>>) {
		use super::common::Message::Extra;

		self.agent = Agent::bridge(link.callback(|msg| match msg {
			Response::State(s) => Extra(FormMessage::SetState(s)),
			Response::RenderQuoted(req) => Extra(FormMessage::QuotePost(req)),
		}))
		.into();
	}

	fn update_extra(
		&mut self,
		link: &ComponentLink<PostCommon<Self>>,
		msg: Self::MessageExtra,
	) -> bool {
		util::with_logging(|| {
			use FormMessage::*;

			Ok(match msg {
				SetState(s) => {
					self.state = s;
					true
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
						link,
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
						self.state != State::Draft || old.is_empty(),
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
							link,
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

	fn render_id<'s, 'c>(&self, _: &RenderCtx<'s, 'c, Self>) -> String {
		"post-form".into()
	}

	fn render_body<'s, 'c>(&self, _: &RenderCtx<'s, 'c, Self>) -> Html {
		html! {
			<textarea ref=self.text_area.clone()>
				{"TODO: body updates"}
			</textarea>
		}
	}

	fn render_after<'s, 'c>(&self, _: &RenderCtx<'s, 'c, Self>) -> Html {
		html! {
			<span>{"TODO: controls"}</span>
		}
	}
}

impl Inner {
	// Return input textarea element
	fn text_area(&self) -> util::Result<web_sys::HtmlTextAreaElement> {
		match self.text_area.get() {
			Some(el) => {
				el.dyn_into().map_err(|_| "not a textarea element".into())
			}
			None => Err("no textarea found".into()),
		}
	}

	// Replace the current body and set the cursor to the input's end.
	// commit: commit any changes to the server
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

	// Send message to execute on the next animation frame
	fn on_next_frame(
		&mut self,
		link: &ComponentLink<PostCommon<Self>>,
		msg: FormMessage,
	) {
		self.render_task =
			RenderService::new()
				.request_animation_frame(link.callback(move |_| {
					super::common::Message::Extra(msg.clone())
				}))
				.into();
	}

	// Commit body changes to server
	fn commit(&mut self, body: String) {
		self.agent.as_mut().unwrap().send(Request::CommitText(body));
	}

	// Resize textarea to content width and adjust height
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

// State oif the agent FSM
#[derive(Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum State {
	// Ready to create posts
	Ready,

	// Post creation controls locked
	Locked,

	// Sent a request to allocate a post
	Allocating,

	// Post open and allocated to the server
	Allocated,

	// Captcha solution required to proceed.
	// This can only take place as an interrupt from the server during
	// Allocating.
	NeedCaptcha,

	// Allocated post during loss of connectivity
	Stalled,

	// Post open but not yet allocating
	Draft,

	// Suffered unrecoverable error
	Erred,
}

impl Default for State {
	fn default() -> Self {
		Self::Locked
	}
}

pub enum Message {
	ConnStateUpdate(connection::State),
	SelectionChange,
}

// Currently selected text and elements
struct Selection {
	start: web_sys::Element,
	end: web_sys::Element,
	text: String,
}

#[derive(Serialize, Deserialize)]
pub enum Request {
	// Quote a post and any selected text
	QuotePost { post: u64, el_id: String },

	// Register as a post form view
	SubViewUpdates,

	// Commit text body changes
	CommitText(String),

	// Open OP for editing. Accepts post ID.
	OpenOPForm(u64),
}

enum Subscription {
	StateOnly,
	ViewUpdates,
}

#[derive(Serialize, Deserialize)]
pub enum Response {
	// Agent state update
	State(State),

	// Render quoted post in view
	RenderQuoted(PostQuoteReq),
}

// Only one PostForm can exist at a time so this agent manages it
pub struct Agent {
	state: State,
	link: AgentLink<Self>,
	subscribers: HashMap<HandlerId, Subscription>,

	#[allow(unused)]
	conn: Box<dyn Bridge<connection::Connection>>,

	conn_state: connection::State,

	// Store last selected range, so we can access it after a mouse click on
	// quote links, which cause that link to become selected
	last_selection: Option<Selection>,

	// Current state of the open post body text
	post_body: Vec<char>,
}

impl yew::agent::Agent for Agent {
	type Reach = yew::agent::Context;
	type Message = Message;
	type Input = Request;
	type Output = Response;

	fn create(link: AgentLink<Self>) -> Self {
		util::add_static_listener(
			util::document(),
			"selectionchange",
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
			link,
			state: State::Locked,
			conn_state: connection::State::Loading,
			subscribers: Default::default(),
			last_selection: Default::default(),
			post_body: Default::default(),
		}
	}

	fn update(&mut self, msg: Self::Message) {
		use connection::State as CS;
		use Message::*;
		use State as S;

		match msg {
			ConnStateUpdate(cs) => {
				self.conn_state = cs;
				match self.state {
					S::Allocating => match cs {
						CS::Synced | CS::Syncing => (),
						_ => self.set_state(S::Draft),
					},
					S::Allocated => match cs {
						CS::Syncing => (),
						CS::Synced => {
							// TODO: resend body and try to resend any missing
							// buffered image, if a disconnect happened
						}
						_ => self.set_state(S::Stalled),
					},
					S::Stalled => match cs {
						CS::Synced => {
							// TODO: resend body and try to resend any buffered
							// image, if none yet set
							self.set_state(S::Allocated);
						}
						_ => (),
					},
					S::Draft => match cs {
						CS::Synced => self.commit_pending(),
						_ => (),
					},
					S::NeedCaptcha => match cs {
						CS::Synced => (),
						_ => self.set_state(S::Draft),
					},
					_ => (),
				}
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
			QuotePost { post, el_id } => {
				self.quote_post(post, el_id);
			}
			SubViewUpdates => {
				self.subscribers.insert(h, Subscription::ViewUpdates);
			}
			CommitText(new) => self.commit_text(new.chars().collect()),
			OpenOPForm(id) => {
				// TODO: hide post currently being edited

				todo!()
			}
		}
	}
}

impl Agent {
	// Set new state and send it to all subscribers
	fn set_state(&mut self, new: State) {
		if self.state != new {
			self.state = new;
			if self.state == State::Allocated {
				self.commit_pending();
			}
			for id in self.subscribers.keys() {
				self.send_current_state(*id);
			}
		}
	}

	fn send_current_state(&self, subscriber: HandlerId) {
		self.link.respond(subscriber, Response::State(self.state));
	}

	// Try allocating a post, if it is eligible and not yet allocated.
	// Returns, if a post is allocated or allocating.
	fn try_alloc(&mut self) -> bool {
		use State::*;

		match self.state {
			Ready | Draft => state::read(|s| {
				if !s.location.is_thread() {
					return false;
				}

				connection::send(
					MessageType::InsertPost,
					&protocol::payloads::PostCreationReq {
						sage: s.new_post_opts.sage,
						thread: s.location.feed.as_u64(),
						opts: protocol::payloads::NewPostOpts {
							name: s.new_post_opts.name.clone(),
						},
					},
				);
				self.set_state(Allocating);
				true
			}),
			Allocating | Allocated | Stalled => true,
			_ => false,
		}
	}

	fn quote_post(&mut self, post: u64, el_id: String) {
		util::with_logging(|| {
			if !self.try_alloc() {
				return Ok(());
			}

			let sel_text = match (
				&util::document().get_element_by_id(&el_id),
				&self.last_selection,
			) {
				(Some(target_post), Some(sel)) => {
					// Check, if selection bound is mid-post
					let in_middle =
						|el: &web_sys::Element| -> util::Result<bool> {
							Ok(el.closest("blockquote")?.is_some()
								&& el
									.closest("article")?
									.map(|el| {
										el.is_same_node(target_post.dyn_ref())
									})
									.unwrap_or(false))
						};

					if (in_middle(&sel.start)?
						|| sel.start.is_same_node(target_post.dyn_ref()))
						&& (in_middle(&sel.end)?
							|| match (
								sel.end.closest("article")?,
								target_post.next_sibling(),
							) {
								(Some(a), Some(b)) => a.is_same_node(Some(&b)),
								_ => false,
							}) {
						sel.text.clone()
					} else {
						Default::default()
					}
				}
				_ => Default::default(),
			};

			for h in self
				.subscribers
				.iter()
				.filter(|(_, s)| matches!(s, Subscription::ViewUpdates))
			{
				self.link.respond(
					*h.0,
					Response::RenderQuoted(PostQuoteReq {
						id: post,
						text: sel_text.clone(),
					}),
				)
			}

			Ok(())
		});
	}

	// Diff and commit text changes to server
	fn commit_text(&mut self, new: Vec<char>) {
		if !self.try_alloc() || self.state != State::Allocated {
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
			// Send a message removing the last character of the line to
			connection::send(MessageType::Backspace, &());
		} else {
			// Commit any other text body change that is not an append or
			// backspace
			connection::send(
				MessageType::PatchPostBody,
				&protocol::payloads::post_body::TextPatch::new(
					&self.post_body,
					&new,
				),
			);
		}

		self.post_body = new;
	}

	// Commit any pending text or images
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
