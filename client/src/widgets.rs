use yew::{html, Component, ComponentLink, Html, InputData, Properties};

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
	InputTag(usize, String),
	RemoveTag(usize),
	AddTag,
}

impl Component for NewThreadForm {
	type Message = Msg;
	type Properties = ();

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			link: link,
			expanded: false,
			available_tags: vec![],
			selected_tags: vec!["".into()],
		}

		// TODO: Fetch tag list from DB
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Msg::Toggle(expand) => {
				self.expanded = expand;
				true
			}
			Msg::InputTag(i, val) => {
				if let Some(t) = self.selected_tags.get_mut(i) {
					*t = val;
				}
				false
			}
			Msg::RemoveTag(i) => {
				if self.selected_tags.len() == 1 {
					self.selected_tags[0].clear();
				} else {
					self.selected_tags = self
						.selected_tags
						.iter()
						.enumerate()
						.filter(|(j, _)| *j != i)
						.map(|(_, s)| s.clone())
						.collect();
				}
				true
			}
			Msg::AddTag => {
				if self.selected_tags.len() < 3 {
					self.selected_tags.push("".into());
				}
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
						style="width: 100%"
					/>
					<hr></hr>
					{self.render_tags()}
					<hr></hr>
					<span>
						<input
							type="submit"
							style="width: 50%"
						/>
						<input
							type="button"
							value={localize!("cancel")}
							style="width: 50%"
							onclick={self.link.callback(|_| Msg::Toggle(false))}
						/>
					</span>
					<datalist id="available-tags">
						{
							for self
								.available_tags
								.iter()
								.filter(|t|
									!self.selected_tags.iter().any(|s| &s == t)
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
		if v.len() < 3 {
			// TODO: Click handler
			v.push(html! {
				<input
					type="button"
					value={localize!("add_tag")}
					onclick={self.link.callback(|_| Msg::AddTag)}
				>
				</input>
			});
		}
		v.into_iter().collect()
	}

	fn render_tag(&self, tag: &str, id: usize) -> Html {
		html! {
			<span>
				<input
					placeholder={localize!{"tag"}}
					required=true
					type="text"
					maxlength="20"
					minlength="1"
					value={tag}
					list="available-tags"
					oninput={
						self.link
						.callback(move |e: InputData|
							Msg::InputTag(id, e.value)
						)
					}
				/>
				<a
					class="act"
					onclick={self.link.callback(move |_| Msg::RemoveTag(id))}
				>
					{"X"}
				</a>
			</span>
		}
	}
}
