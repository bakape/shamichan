use super::{buttons, comp_util, post::posting, state};
use std::collections::HashSet;
use yew::{html, Html, Properties};

/// Central thread container
pub type Thread = comp_util::HookedComponent<Inner>;

#[derive(Default)]
pub struct Inner {}

/// Posts to display in a thread
#[derive(Clone, Eq, PartialEq, Debug)]
pub enum PostSet {
	/// Display OP + last 5 posts
	Last5Posts,

	/// Display OP + selected pages
	Pages(HashSet<u32>),
}

impl Default for PostSet {
	fn default() -> Self {
		Self::Last5Posts
	}
}

#[derive(Clone, Properties, Eq, PartialEq, Debug)]
pub struct Props {
	pub id: u64,
	pub pages: PostSet,
}

impl comp_util::Inner for Inner {
	type Message = ();
	type Properties = Props;

	fn update_message() -> Self::Message {
		()
	}

	fn subscribe_to(props: &Self::Properties) -> Vec<state::Change> {
		vec![state::Change::Thread(props.id)]
	}

	fn update(
		&mut self,
		_: &mut comp_util::Ctx<Self>,
		_: Self::Message,
	) -> bool {
		true
	}

	fn view(&self, c: &comp_util::Ctx<Self>) -> Html {
		use super::post::ThreadPost;
		use PostSet::*;

		// TODO: Filter hidden posts
		let posts: Vec<u64> = state::read(|s| match &c.props().pages {
			Last5Posts => {
				let mut v = Vec::with_capacity(5);
				let page_count = s
					.threads
					.get(&c.props().id)
					.map(|t| t.page_count)
					.unwrap_or(1);
				self.read_page_posts(&mut v, c.props().id, page_count - 1, s);
				if v.len() < 5 && page_count > 1 {
					self.read_page_posts(
						&mut v,
						c.props().id,
						page_count - 2,
						s,
					);
				}
				v.sort_unstable();
				if v.len() > 5 {
					v[v.len() - 5..].iter().copied().collect()
				} else {
					v
				}
			}
			Pages(pages) => {
				let mut v = Vec::with_capacity(300);
				for p in pages.iter() {
					self.read_page_posts(&mut v, c.props().id, *p, s);
				}
				v.sort_unstable();
				v
			}
		});

		html! {
			<section class="thread-container" key=c.props().id>
				<ThreadPost id=c.props().id />
				{
					for posts.into_iter().map(|id| {
						html! {
							<ThreadPost id=id />
						}
					})
				}
			</section>
		}
	}
}

impl Inner {
	/// Read the post IDs of a page, excluding the OP, into dst
	fn read_page_posts(
		&self,
		dst: &mut Vec<u64>,
		thread: u64,
		page: u32,
		s: &state::State,
	) {
		if let Some(posts) = s.posts_by_thread_page.get_by_key(&(thread, page))
		{
			dst.extend(posts.iter().filter(|id| **id != thread));
		}
	}
}

#[derive(Properties, Eq, PartialEq, Clone)]
struct ReplyProps {
	thread: u64,
}

struct ReplyButton {
	props: ReplyProps,
	link: yew::ComponentLink<Self>,
	posting: Box<dyn yew::agent::Bridge<posting::Agent>>,
	state: posting::State,
}

enum ReplyMessage {
	SetState(posting::State),
	Clicked,
	NOP,
}

impl yew::Component for ReplyButton {
	super::comp_prop_change! {ReplyProps}
	type Message = ReplyMessage;

	fn create(props: Self::Properties, link: yew::ComponentLink<Self>) -> Self {
		use yew::agent::Bridged;

		Self {
			props,
			posting: posting::Agent::bridge(link.callback(|msg| match msg {
				posting::Response::State(s) => ReplyMessage::SetState(s),
				_ => ReplyMessage::NOP,
			})),
			link,
			state: Default::default(),
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		use ReplyMessage::*;

		match msg {
			SetState(s) => {
				self.state = s;
				true
			}
			Clicked => {
				if self.state == posting::State::Ready {
					self.posting
						.send(posting::Request::OpenDraft(self.props.thread))
				}
				false
			}
			NOP => false,
		}
	}

	fn view(&self) -> yew::Html {
		use posting::State::*;

		match self.state {
			Ready | Locked => html! {
				<buttons::AsideButton
					text="reply"
					disabled=matches!(self.state, Locked),
					on_click=self.link.callback(|e: yew::events::MouseEvent| {
						if e.button() == 0 {
							ReplyMessage::Clicked
						} else {
							ReplyMessage::NOP
						}
					})
				/>
			},
			_ => html! {},
		}
	}
}
