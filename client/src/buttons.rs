use yew::{html, Callback, Component, ComponentLink, Html, Properties};

#[derive(Clone, Properties, PartialEq)]
pub struct Props {
	pub text: &'static str,

	#[prop_or_default]
	pub disabled: bool,

	pub on_click: Callback<yew::events::MouseEvent>,
	//
	// TODO: Optional middle click handler for opening in a new tab
}

macro_rules! impl_button {
	($name:ident, $view:expr) => {
		pub struct $name {
			props: Props,
		}

		impl Component for $name {
			comp_no_update! {}
			comp_prop_change! {Props}

			fn create(props: Self::Properties, _: ComponentLink<Self>) -> Self {
				Self { props }
			}

			fn view(&self) -> Html {
				$view(&self.props)
			}
		}
	};
}

impl_button! { Anchor, |props: &Props| {
	html! {
		<a onclick=props.on_click.clone()>
			{localize!(props.text)}
		</a>
	}
}}

impl_button! { SpanButton, |props: &Props| {
	html! {
		<span class="act">
			<Anchor
				on_click=props.on_click.clone()
				text=props.text
			/>
		</span>
	}
}}

impl_button! { AsideButton, |props: &Props| {
	html! {
		<aside class="act glass" >
			<Anchor
				on_click=props.on_click.clone()
				text=props.text
			/>
		</aside>
	}
}}
