use yew::{html, Component, ComponentLink, Html, Properties};

pub struct AsideRow {
	props: Props,
}

#[derive(Clone, Properties)]
pub struct Props {
	pub is_top: bool,
}

impl Component for AsideRow {
	type Message = ();
	type Properties = Props;

	fn create(props: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self { props: props }
	}

	fn update(&mut self, _: Self::Message) -> bool {
		false
	}

	fn view(&self) -> Html {
		// TODO: Routing + switch on page type

		html! {
			<>
				<span
					class="aside-container"
					style={
						if self.props.is_top {
							"margin-top: 1.5em;"
						} else {
							""
						}
					}
				>
					{
						if self.props.is_top {
							html! {
								<aside id="thread-form-container">
									<span class="act">
										<a class="new-thread-button">
											{localize!("new_thread")}
										</a>
									</span>
								</aside>
							}
						} else {
							html! {}
						}
					}
					<aside class="act glass">
						<a href="catalog">{localize!("catalog")}</a>
					</aside>
				</span>
			</>
		}
	}
}
