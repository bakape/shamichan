use super::state;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html, Properties};

// Central thread container
pub struct Post {
	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,

	#[allow(unused)]
	link: ComponentLink<Self>,

	id: u64,
}

pub enum Message {
	PostChange,
	NOP,
}

#[derive(Clone, Properties)]
pub struct Props {
	pub id: u64,
}

impl Component for Post {
	type Message = Message;
	type Properties = Props;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut s = state::Agent::bridge(link.callback(|u| match u {
			state::Subscription::PostChange(_) => Message::PostChange,
			_ => Message::NOP,
		}));
		s.send(state::Request::Subscribe(state::Subscription::PostChange(
			props.id,
		)));
		Self {
			id: props.id,
			state: s,
			link: link,
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::PostChange => true,
			Message::NOP => false,
		}
	}

	fn view(&self) -> Html {
		html! {
			<>
				<article id={format!("p-{}", self.id)}>
				</article>
			</>
		}
	}
}
