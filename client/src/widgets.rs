use crate::{buttons::AsideButton, connection, state};
use yew::{
	agent::{Bridge, Bridged},
	html,
	services::fetch::FetchTask,
	Component, ComponentLink, Html, InputData, NodeRef, Properties,
};

pub struct AsideRow {
	link: ComponentLink<Self>,
	props: Props,

	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,
}

#[derive(Clone, Properties)]
pub struct Props {
	#[prop_or_default]
	pub is_top: bool,
}

pub enum Message {
	FeedChange,
	NOP,
}

impl Component for AsideRow {
	type Message = Message;
	type Properties = Props;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		use state::{Agent, Request, Response, Subscription};

		let mut s = Agent::bridge(link.callback(|u| match u {
			Response::LocationChange { old, new } => {
				if old.feed != new.feed {
					Message::FeedChange
				} else {
					Message::NOP
				}
			}
			_ => Message::NOP,
		}));
		s.send(Request::Subscribe(Subscription::LocationChange));
		Self {
			props,
			link,
			state: s,
		}
	}

	fn update(&mut self, _: Self::Message) -> bool {
		false
	}

	fn view(&self) -> Html {
		use state::FeedID;

		let feed = &state::get().location.feed;
		let is_thread = matches!(feed, FeedID::Thread(_));

		html! {
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
					if !is_thread && self.props.is_top {
						html! {
							<NewThreadForm />
						}
					} else {
						html! {}
					}
				}
				// TODO: swap between index and catalog. Persist last mode to
				// local storage.
				<AsideButton
					text=if is_thread {
						"return"
					} else {
						"catalog"
					}
					on_click=self.link.callback(|_| Message::NOP)
				/>
				{
					match feed {
						FeedID::Thread(f) => html! {
							<crate::page_selector::PageSelector thread=f.id />
						},
						_ => html! {},
					}
				}
			</span>
		}
	}
}

struct NewThreadForm {
	el: NodeRef,
	link: ComponentLink<Self>,
	expanded: bool,
	available_tags: Vec<String>,
	selected_tags: Vec<String>,
	conn_state: connection::State,

	#[allow(unused)]
	fetch_task: Option<FetchTask>,
	#[allow(unused)]
	conn: Box<dyn Bridge<connection::Connection>>,
}

enum Msg {
	Toggle(bool),
	InputTag(usize, String),
	RemoveTag(usize),
	AddTag,
	Submit,
	ConnState(connection::State),
	FetchedUsedTags(Vec<String>),
	NOP,
}

impl Component for NewThreadForm {
	type Message = Msg;
	type Properties = ();

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		use yew::format::{Json, Nothing};
		use yew::services::fetch::{FetchService, Request, Response};

		Self {
			conn: connection::Connection::bridge(
				link.callback(|s| Msg::ConnState(s)),
			),
			el: NodeRef::default(),
			fetch_task: FetchService::new()
				.fetch(
					Request::get("/api/json/used-tags").body(Nothing).unwrap(),
					link.callback(
						|res: Response<
							Json<Result<Vec<String>, anyhow::Error>>,
						>| match res.into_body() {
							Json(Ok(tags)) => Msg::FetchedUsedTags(tags),
							_ => Msg::NOP,
						},
					),
				)
				.ok(),
			link,
			expanded: false,
			available_tags: vec![],
			selected_tags: vec!["".into()],
			conn_state: connection::State::Loading,
		}
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
				use web_sys::{FormData, HtmlFormElement};

				let f = FormData::new_with_form(
					&(self.el.cast::<HtmlFormElement>()).unwrap(),
				)
				.unwrap();
				connection::send(
					protocol::MessageType::CreateThread,
					&protocol::ThreadCreationReq {
						subject: f
							.get("subject")
							.as_string()
							.unwrap_or_default(),
						tags: f
							.get_all("tag")
							.iter()
							.filter_map(|t| t.as_string())
							.collect(),
						// TODO
						captcha_solution: vec![],
					},
				);

				false
			}
			Msg::ConnState(s) => {
				self.conn_state = s;
				true
			}
			Msg::NOP => false,
			Msg::FetchedUsedTags(tags) => {
				self.available_tags = tags;
				true
			}
		}
	}

	fn view(&self) -> Html {
		html! {
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
		}
	}
}

impl NewThreadForm {
	fn render_form(&self) -> Html {
		html! {
			<form
				id="new-thread-form"
				ref=self.el.clone()
				style="display: flex; flex-direction: column;"
				onsubmit={self.link.callback(|e: yew::events::Event| {
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
						disabled=self.conn_state != connection::State::Synced
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
				/>
			});
		}
		v.into_iter().collect()
	}

	fn render_tag(&self, tag: &str, id: usize) -> Html {
		html! {
			<span>
				<input
					placeholder=localize!("tag")
					required=true
					type="text"
					maxlength="20"
					minlength="1"
					value=tag
					name="tag"
					list="available-tags"
					oninput=self.link.callback(move |e: InputData|
						Msg::InputTag(id, e.value)
					)
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
