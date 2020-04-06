use super::state;
use state::Thread;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html};

// Central thread container
pub struct Threads {
	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,

	#[allow(unused)]
	link: ComponentLink<Self>,
}

impl Component for Threads {
	type Message = bool;
	type Properties = ();

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		use state::{Agent, Request, Response, Subscription};

		let mut s = Agent::bridge(link.callback(|u| {
			matches!(
				u,
				Response::NoPayload(Subscription::ThreadListChange)
					| Response::LocationChange{..}
			)
		}));
		s.send(Request::Subscribe(Subscription::ThreadListChange));
		s.send(Request::Subscribe(Subscription::ThreadListChange));
		Self {
			state: s,
			link: link,
		}
	}

	fn update(&mut self, rerender: Self::Message) -> bool {
		rerender
	}

	fn view(&self) -> Html {
		use super::thread as view;
		use state::FeedID;

		let s = state::get();
		match &s.location.feed {
			FeedID::Index => {
				let mut threads: Vec<&Thread> = s.threads.values().collect();
				// TODO: Different sort orders
				threads
					.sort_unstable_by_key(|t| std::cmp::Reverse(t.bumped_on));

				let mut w = Vec::with_capacity(threads.len() * 2);
				for (i, t) in threads.into_iter().enumerate() {
					if i != 0 {
						w.push(html! {
							<hr />
						});
					}
					w.push(html! {
						<view::Thread id=t.id pages=view::PostSet::Last5Posts />
					});
				}

				html! {
					<section>
						{w.into_iter().collect::<Html>()}
					</section>
				}
			}
			FeedID::Catalog => html! {
				<span>{"TODO"}</span>
			},
			FeedID::Thread { id, page } => {
				html! {
					<view::Thread
						id=id
						pages=view::PostSet::Page(*page as u32)
					/>
				}
			}
		}
	}
}
