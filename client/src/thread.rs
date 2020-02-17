use super::state;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html, Properties};

// Central thread container
pub struct Thread {
	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,

	#[allow(unused)]
	link: ComponentLink<Self>,

	id: u64,
	pages: PageSet,
}

pub enum Message {
	ThreadChange,
	NOP,
}

// Pages to display in a thread
#[derive(Clone)]
pub enum PageSet {
	// Display OP + last 5 posts
	Last5Posts,

	// Display OP + selected pages.
	// If page set is smaller than 3, insert zeroes.
	Pages([u32; 3]),
}

impl Default for PageSet {
	fn default() -> Self {
		Self::Last5Posts
	}
}

#[derive(Clone, Properties)]
pub struct Props {
	#[props(required)]
	pub id: u64,
	#[props(required)]
	pub pages: PageSet,
}

impl Component for Thread {
	type Message = Message;
	type Properties = Props;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut s = state::Agent::bridge(link.callback(|u| match u {
			state::Subscription::ThreadChange(_) => Message::ThreadChange,
			_ => Message::NOP,
		}));
		s.send(state::Request::Subscribe(
			state::Subscription::ThreadChange(props.id),
		));
		Self {
			id: props.id,
			pages: props.pages,
			state: s,
			link: link,
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::ThreadChange => true,
			Message::NOP => false,
		}
	}

	fn view(&self) -> Html {
		let s = state::get();
		// TODO: Filter hidden posts
		let posts: Vec<u64> = match s.posts_by_thread.get(&self.id) {
			Some(set) => match self.pages {
				PageSet::Last5Posts => {
					let mut v: Vec<u64> = set
						.iter()
						.filter(|id| **id != self.id)
						.copied()
						.collect();
					v.sort_unstable();
					if v.len() > 5 {
						v[v.len() - 5..].into_iter().copied().collect()
					} else {
						v
					}
				}
				PageSet::Pages(pages) => {
					let mut v: Vec<u64> = set
						.iter()
						.filter(|id| {
							**id != self.id
								&& match s.posts.get(*id) {
									Some(p) => {
										pages.iter().any(|page| *page == p.page)
									}
									None => false,
								}
						})
						.copied()
						.collect();
					v.sort_unstable();
					v
				}
			},
			None => vec![],
		};

		html! {
			<>
				<section class="thread-container">
					<super::post::Post id={self.id} />
					{
						for posts.into_iter().map(|id| {
							html! {
								<super::post::Post id={id} />
							}
						})
					}
					<aside>
						{"TODO: Reply"}
					</aside>
				</section>
			</>
		}
	}
}
