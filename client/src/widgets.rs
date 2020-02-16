use super::connection;
use super::util;
use stdweb::web::event::{IEvent, SubmitEvent};
use stdweb::web::{Element, FormData, FormDataEntry};
use yew::agent::{Bridge, Bridged};
use yew::{
	html, Component, ComponentLink, Html, InputData, NodeRef, Properties,
};

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
								<NewThreadForm />
							}
						} else {
							html! {}
						}
					}
					<aside class="act glass">
						<a>{localize!("catalog")}</a>
					</aside>
				</span>
			</>
		}
	}
}

struct NewThreadForm {
	el: NodeRef,
	link: ComponentLink<Self>,
	expanded: bool,
	available_tags: Vec<String>,
	selected_tags: Vec<String>,
	conn: Box<dyn Bridge<connection::Connection>>,
	conn_state: connection::State,
}

enum Msg {
	Toggle(bool),
	InputTag(usize, String),
	RemoveTag(usize),
	AddTag,
	Submit,
	ConnState(connection::State),
}

impl Component for NewThreadForm {
	type Message = Msg;
	type Properties = ();

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			conn: connection::Connection::bridge(
				link.callback(|s| Msg::ConnState(s)),
			),
			el: NodeRef::default(),
			link: link,
			expanded: false,
			available_tags: vec![],
			selected_tags: vec!["".into()],
			conn_state: connection::State::Loading,
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
			Msg::Submit => {
				util::with_logging(|| {
					let f = FormData::from_element(
						&(self.el.try_into::<Element>()).unwrap(),
					)
					.unwrap();
					self.conn.send(super::encode_message!(
						protocol::MessageType::CreateThread,
						&protocol::ThreadCreationReq {
							subject: match f.get("subject") {
								Some(FormDataEntry::String(s)) => s,
								_ => "".into(),
							},
							tags: f
								.get_all("tag")
								.into_iter()
								.filter_map(|t| match t {
									FormDataEntry::String(s) => Some(s),
									_ => None,
								})
								.collect(),
							// TODO
							captcha_solution: vec![],
						}
					)?);
					Ok(())
				});
				false
			}
			Msg::ConnState(s) => {
				self.conn_state = s;
				true
			}
		}
	}

	fn view(&self) -> Html {
		html! {
			<>
				<aside id="thread-form-container">
					<span class={if !self.expanded { "act" } else { "" }}>
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
					ref=self.el.clone()
					style="display: flex; flex-direction: column;"
					onsubmit={self.link.callback(|e: SubmitEvent| {
						e.prevent_default();
						Msg::Submit
					})}
				>
					<input
						placeholder=localize!{"subject"}
						name="subject"
						required=true
						type="text"
						maxlength="100"
						style="width: 100%"
					/>
					<hr />
					{self.render_tags()}
					<hr />
					<span>
						<input
							type="submit"
							style="width: 50%"
							disabled={
								self.conn_state != connection::State::Synced
							}
						/>
						<input
							type="button"
							value=localize!("cancel")
							style="width: 50%"
							onclick=self.link.callback(|_| Msg::Toggle(false))
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
										<option value=t></option>
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
			v.push(html! {
				<input
					type="button"
					value=localize!("add_tag")
					onclick=self.link.callback(|_| Msg::AddTag)
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
					placeholder=localize!{"tag"}
					required=true
					type="text"
					maxlength="20"
					minlength="1"
					value={tag}
					name="tag"
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
					onclick=self.link.callback(move |_| Msg::RemoveTag(id))
				>
					{"X"}
				</a>
			</span>
		}
	}
}
