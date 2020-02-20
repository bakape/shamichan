use yew::{html, Component, ComponentLink, Html};

pub struct Background {}

impl Component for Background {
	type Message = ();
	type Properties = ();

	fn create(_: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self {}
	}

	fn update(&mut self, _: Self::Message) -> bool {
		false
	}

	fn view(&self) -> Html {
		html! {
			<div id="user-background"></div>
		}
	}
}
