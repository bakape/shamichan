use yew::{html, Callback, Component, ComponentLink, Html, Properties};

#[derive(Clone, Properties)]
pub struct Props {
	pub text: &'static str,
	pub on_click: Callback<yew::events::MouseEvent>,
}

pub struct Anchor {
	props: Props,
}

impl Component for Anchor {
	type Message = ();
	type Properties = Props;

	fn create(props: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self { props }
	}

	fn update(&mut self, _: Self::Message) -> bool {
		false
	}

	fn view(&self) -> Html {
		html! {
			<a onclick=self.props.on_click.clone()>
				{localize!(self.props.text)}
			</a>
		}
	}
}

pub struct SpanButton {
	props: Props,
}

impl Component for SpanButton {
	type Message = ();
	type Properties = Props;

	fn create(props: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self { props }
	}

	fn update(&mut self, _: Self::Message) -> bool {
		false
	}

	fn view(&self) -> Html {
		html! {
			<span class="act">
				<Anchor
					on_click=self.props.on_click.clone()
					text=self.props.text
				/>
			</span>
		}
	}
}

pub struct AsideButton {
	props: Props,
}

impl Component for AsideButton {
	type Message = ();
	type Properties = Props;

	fn create(props: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self { props }
	}

	fn update(&mut self, _: Self::Message) -> bool {
		false
	}

	fn view(&self) -> Html {
		html! {
			<aside class="act glass" >
				<Anchor
					on_click=self.props.on_click.clone()
					text=self.props.text
				/>
			</aside>
		}
	}
}
