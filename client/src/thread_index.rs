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

pub enum Message {
	ThreadListChange,
	NOP,
}

impl Component for Threads {
	type Message = Message;
	type Properties = ();

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut s = state::Agent::bridge(link.callback(|u| match u {
			state::Subscription::ThreadListChange => Message::ThreadListChange,
			_ => Message::NOP,
		}));
		s.send(state::Request::Subscribe(
			state::Subscription::ThreadListChange,
		));
		Self {
			state: s,
			link: link,
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::ThreadListChange => true,
			Message::NOP => false,
		}
	}

	fn view(&self) -> Html {
		use super::thread as view;

		// TODO: Routing + switch on page type

		let mut threads: Vec<&Thread> = state::get().threads.values().collect();
		// TODO: Different sort orders
		threads.sort_unstable_by_key(|t| std::cmp::Reverse(t.bumped_on));

		let mut w = Vec::with_capacity(threads.len() * 2);
		for (i, t) in threads.into_iter().enumerate() {
			if i != 0 {
				w.push(html! {
					<hr />
				});
			}
			w.push(html! {
				<view::Thread id=t.id pages=view::PageSet::Last5Posts />
			});
		}

		html! {
			<section>
				{w.into_iter().collect::<Html>()}
			</section>
		}
	}
}
