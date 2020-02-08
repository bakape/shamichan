use yew::{html, Component, ComponentLink, Html};

// Central thread container
pub struct Threads {}

impl Component for Threads {
	type Message = ();
	type Properties = ();

	fn create(_: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self {}
	}

	fn update(&mut self, _: Self::Message) -> bool {
		false
	}

	fn view(&self) -> Html {
		// TODO: Routing + switch on page type

		html! {
			<>
				<section>{"TODO"}</section>
			</>
		}
	}
}
