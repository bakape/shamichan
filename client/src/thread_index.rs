use super::state;
use state::Thread;
use yew::{html, Component, ComponentLink, Html};

// Central thread container
pub struct Threads {
	#[allow(unused)]
	bridge: state::HookBridge,

	#[allow(unused)]
	link: ComponentLink<Self>,
}

impl Component for Threads {
	type Message = ();
	type Properties = ();

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			bridge: state::hook(&link, &[state::Change::ThreadList], |_| ()),
			link: link,
		}
	}

	fn update(&mut self, _: Self::Message) -> bool {
		true
	}

	fn view(&self) -> Html {
		use super::thread as view;
		use state::FeedID;

		state::read(|s| {
			match &s.location.feed {
				FeedID::Catalog => {
					html! {
						<span>{"TODO"}</span>
					}
				}
				FeedID::Index => {
					let mut threads: Vec<&Thread> =
						s.threads.values().collect();
					// TODO: Different sort orders
					threads.sort_unstable_by_key(|t| {
						std::cmp::Reverse(t.bumped_on)
					});

					let mut w = Vec::with_capacity(threads.len() * 2);
					for (i, t) in threads.into_iter().enumerate() {
						if i != 0 {
							w.push(html! {
								<hr />
							});
						}
						w.push(html! {
							<view::Thread
								id=t.id pages=view::PostSet::Last5Posts
							/>
						});
					}

					html! {
						<section>
							{w.into_iter().collect::<Html>()}
						</section>
					}
				}
				FeedID::Thread { id, page } => {
					html! {
						<view::Thread
							id=id
							pages=view::PostSet::Page(*page as u32)
						/>
					}
				}
			}
		})
	}
}
