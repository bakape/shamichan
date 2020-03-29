use super::buttons;
use super::state;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html, Properties};

// Central thread container
pub struct Thread {
	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,

	#[allow(unused)]
	link: ComponentLink<Self>,

	id: u64,
	pages: PostSet,
}

pub enum Message {
	ThreadChange,
	NOP,
}

// Posts to display in a thread
#[derive(Clone)]
pub enum PostSet {
	// Display OP + last 5 posts
	Last5Posts,

	// Display OP + selected page
	Page(u32),
}

impl Default for PostSet {
	fn default() -> Self {
		Self::Last5Posts
	}
}

#[derive(Clone, Properties)]
pub struct Props {
	pub id: u64,
	pub pages: PostSet,
}

impl Component for Thread {
	type Message = Message;
	type Properties = Props;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		use state::{Agent, Request, Response, Subscription};

		let mut s = Agent::bridge(link.callback(|u| match u {
			Response::NoPayload(Subscription::ThreadChange(_)) => {
				Message::ThreadChange
			}
			_ => Message::NOP,
		}));
		s.send(Request::Subscribe(Subscription::ThreadChange(props.id)));
		Self {
			id: props.id,
			pages: props.pages,
			state: s,
			link,
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::ThreadChange => true,
			Message::NOP => false,
		}
	}

	fn view(&self) -> Html {
		// TODO: Filter hidden posts

		let posts: Vec<u64> = match self.pages {
			PostSet::Last5Posts => {
				let mut v = Vec::with_capacity(5);
				let page_count =
					state::get().page_counts.get(&self.id).unwrap_or(&1);
				self.read_page_posts(&mut v, page_count - 1);
				if v.len() < 5 && page_count > &1 {
					self.read_page_posts(&mut v, page_count - 2);
				}
				v.sort_unstable();
				if v.len() > 5 {
					v[v.len() - 5..].iter().copied().collect()
				} else {
					v
				}
			}
			PostSet::Page(page) => {
				let mut v = Vec::with_capacity(300);
				self.read_page_posts(&mut v, page);
				v.sort_unstable();
				v
			}
		};

		html! {
			<section class="thread-container">
				<super::post::Post id=self.id />
				{
					for posts.into_iter().map(|id| {
						html! {
							<super::post::Post id={id} />
						}
					})
				}
				// TODO: Reply button that opens a reply creation modal on both
				// the thread index and individual thread pages (allow posting
				// from thread index).
				<buttons::AsideButton
					text="reply"
					on_click=self.link.callback(|_| Message::NOP)
				/>
			</section>
		}
	}
}

impl Thread {
	// Read the post IDs of a page, excluding the OP, into dst
	fn read_page_posts(&self, dst: &mut Vec<u64>, page: u32) {
		if let Some(posts) =
			state::get().posts_by_thread_page.get(&(self.id, page))
		{
			dst.extend(posts.iter().filter(|id| **id != self.id));
		}
	}
}
