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
				<span class="aside-container">
					{
						if self.props.is_top {
							html! {
								<NewThreadForm />
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

struct NewThreadForm {
	link: ComponentLink<Self>,
	expanded: bool,
	available_tags: Vec<String>,
	selected_tags: Vec<String>,
}

enum Msg {
	Toggle(bool),
}

impl Component for NewThreadForm {
	type Message = Msg;
	type Properties = ();

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			link: link,
			expanded: false,
			available_tags: vec![],
			selected_tags: vec![],
		}

		// TODO: Fetch tag list from DB
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Msg::Toggle(expand) => {
				self.expanded = expand;
				true
			}
		}
	}

	fn view(&self) -> Html {
		html! {
			<>
				<aside id="thread-form-container">
					<span
						class={
							if !self.expanded {
								"act"
							} else {
								""
							}
						}
					>
						{
							if self.expanded {
								self.render_form()
							} else {
								html! {
									<a
										class="new-thread-button"
										onclick={
											self.link
											.callback(|_| Msg::Toggle(true))
										}
									>
										{localize!("new_thread")}
									</a>
								}
							}
						}
					</span>
				</aside>
			</>
		}
	}
}

impl NewThreadForm {
	fn render_form(&self) -> Html {
		html! {
			<>
				<form
					id="new-thread-form"
					style="display: flex; flex-direction: column;"
				>
					<input
						placeholder={localize!{"subject"}}
						required=true
						type="text"
						maxlength="100"
					/>
					{self.render_tags()}
					<input type="submit" />
					<datalist id="available-tags">
						{
							for self
								.available_tags
								.iter()
								.filter(|t|
									self.selected_tags.iter().any(|s| &s == t)
								)
								.map(|t| {
									html! {
										<option value={t}></option>
									}
								})
						}
					</datalist>
				</form>
			</>
		}
	}

	fn render_tags(&self) -> Html {
		let mut v = Vec::with_capacity(3);
		for (i, t) in self.selected_tags.iter().enumerate() {
			v.push(self.render_tag(t, i));
		}
		if v.is_empty() {
			v.push(self.render_tag("", 0));
		}
		if v.len() < 3 {
			// TODO: Click handler
			v.push(html! {
				<input type="button" value={localize!("add_tag")}></input>
			});
		}
		v.into_iter().collect()
	}

	fn render_tag(&self, tag: &str, id: usize) -> Html {
		// TODO: Input handler
		html! {
			<input
				placeholder={localize!{"tag"}}
				required=true
				type="text"
				maxlength="100"
				minlength="1"
				value={tag}
				list="available-tags"
				// TODO: Pass tag value and position in message
			/>
		}
	}
}
