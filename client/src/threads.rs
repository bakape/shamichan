use super::state;
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

	fn update(&mut self, _: Self::Message) -> bool {
		true
	}

	fn view(&self) -> Html {
		// TODO: Routing + switch on page type

		html! {
			<>
				<section>{format!("{:?}", state::get().threads)}</section>
			</>
		}
	}
}
