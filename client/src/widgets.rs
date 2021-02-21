use crate::{
	buttons::AsideButton,
	connection,
	post::posting,
	state::{self, FeedID, Focus, Location},
	util,
};
use yew::{
	agent::{Bridge, Bridged},
	html, Component, ComponentLink, Html, InputData, NodeRef, Properties,
};

pub struct AsideRow {
	link: ComponentLink<Self>,
	props: Props,
	app_state: state::StateBridge,
}

#[derive(Clone, Properties, Eq, PartialEq)]
pub struct Props {
	#[prop_or_default]
	pub is_top: bool,
}

impl Component for AsideRow {
	comp_prop_change! {Props}
	type Message = bool;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			app_state: state::hook(
				&link,
				vec![state::Change::Location],
				|| true,
			),
			props,
			link,
		}
	}

	fn update(&mut self, rerender: Self::Message) -> bool {
		rerender
	}

	fn view(&self) -> Html {
		let loc = &self.app_state.get().location;
		let is_thread = loc.is_thread();
		let (label, focus) = if self.props.is_top {
			("bottom", Focus::Bottom)
		} else {
			("top", Focus::Top)
		};

		#[rustfmt::skip]
		macro_rules! navi_button {
			($pat:pat, $label:expr, $loc:expr) => {
				if !matches!(loc.feed, $pat) {
					self.render_navigation_button($label, $loc)
				} else {
					html! {}
				}
			};
		}

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
				{
					self.render_navigation_button(label, Location {
						feed: loc.feed.clone(),
						focus: Some(focus),
					})
				}
				{
					navi_button!(FeedID::Index, "index", Location{
						feed: FeedID::Index,
						focus: None,
					})
				}
				{
					navi_button!(FeedID::Catalog, "catalog", Location{
						feed: FeedID::Catalog,
						focus: None,
					})
				}
				{
					match &loc.feed {
						FeedID::Thread { id, .. } => html! {
							<aside class="glass">
								<crate::page_selector::PageSelector
									thread=id
								/>
							</aside>
						},
						_ => html! {},
					}
				}
			</span>
		}
	}
}

impl AsideRow {
	fn render_navigation_button(
		&self,
		label: &'static str,
		loc: Location,
	) -> Html {
		html! {
			<AsideButton
				text=label
				on_click=self.link.callback(move |_| {
					state::navigate_to(loc.clone());
					false
				})
			/>
		}
	}
}

struct NewThreadForm {
	el: NodeRef,
	link: ComponentLink<Self>,
	expanded: bool,
	selected_tags: Vec<String>,

	sending: bool,

	post_form_state: posting::State,
	conn_state: connection::State,

	#[allow(unused)]
	posting: Box<dyn Bridge<posting::Agent>>,
	#[allow(unused)]
	connection: Box<dyn Bridge<connection::Connection>>,
	app_state: state::StateBridge,
}

enum Msg {
	Toggle(bool),
	InputTag(usize, String),
	RemoveTag(usize),
	AddTag,
	Submit,
	PostFormState(posting::State),
	ConnState(connection::State),
	Rerender,
	NOP,
}

impl Component for NewThreadForm {
	comp_no_props! {}
	type Message = Msg;

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			posting: posting::Agent::bridge(link.callback(|msg| match msg {
				posting::Response::State(s) => Msg::PostFormState(s),
				_ => Msg::NOP,
			})),
			connection: connection::Connection::bridge(
				link.callback(|s| Msg::ConnState(s)),
			),
			el: NodeRef::default(),
			app_state: state::hook(
				&link,
				vec![state::Change::UsedTags],
				|| Msg::Rerender,
			),
			link,
			expanded: false,
			sending: false,
			selected_tags: vec!["".into()],
			post_form_state: Default::default(),
			conn_state: Default::default(),
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Msg::Toggle(expand) => {
				if expand && !self.expanded {
					// Get a fresh list of used thread tags
					connection::send(common::MessageType::UsedTags, &());
				}
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
					self.selected_tags =
						std::mem::take(&mut self.selected_tags)
							.into_iter()
							.enumerate()
							.filter(|(j, _)| *j != i)
							.map(|(_, s)| s)
							.collect();
				}
				true
			}
			Msg::AddTag => {
				if self.selected_tags.len() < 3 {
					self.selected_tags.push("".into());
					true
				} else {
					false
				}
			}
			Msg::Submit => {
				use web_sys::{FormData, HtmlFormElement};

				if self.sending {
					return false;
				}
				self.sending = true;

				if let Err(err) =
					|| -> util::Result {
						let f = FormData::new_with_form(
							&self.el.cast::<HtmlFormElement>().ok_or(
								"could not convert to HtmlFormElement",
							)?,
						)?;

						connection::send(
							common::MessageType::InsertThread,
							&common::payloads::ThreadCreationReq {
								subject: f.get("subject").as_string().ok_or(
									"could not convert subject to string",
								)?,
								tags: f
									.get_all("tag")
									.iter()
									.filter_map(|t| t.as_string())
									.map(|s| s.trim().to_lowercase())
									.into_iter()
									// Ensure lack of duplicates
									.collect::<std::collections::HashSet<_>>()
									.into_iter()
									.collect(),
								opts: common::payloads::NewPostOpts {
									name: self
										.app_state
										.get()
										.new_post_opts
										.name
										.clone(),
								},
								// TODO
								captcha_solution: vec![],
							},
						);
						Ok(())
					}() {
					self.sending = false;
					util::alert(&err);
				}

				true
			}
			Msg::PostFormState(s) => {
				self.post_form_state = s;
				true
			}
			Msg::ConnState(s) => {
				self.conn_state = s;
				true
			}
			Msg::NOP => false,
			Msg::Rerender => true,
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
				onsubmit={self.link.callback(|e: yew::events::FocusEvent| {
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
					pattern=r".*\S+.*"
				/>
				<hr />
				{self.render_tags()}
				<hr />
				<span>
					<input
						type="submit"
						style="width: 50%"
						disabled=self.sending
								|| self.post_form_state
									!= posting::State::Ready
								|| self.conn_state
									!= connection::State::HandshakeComplete
					/>
					<input
						type="button"
						value=localize!("cancel")
						style="width: 50%"
						onclick=self.link.callback(|_| Msg::Toggle(false))
						disabled=self.sending
					/>
				</span>
				<datalist id="used-tags">
					{
						for self.app_state.get().used_tags.clone()
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
		let close_button = self.selected_tags.len() != 1;
		for (i, t) in self.selected_tags.iter().enumerate() {
			v.push(self.render_tag(t, i, close_button));
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

	fn render_tag(&self, tag: &str, id: usize, close_button: bool) -> Html {
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
					list="used-tags"
					pattern=r".*\S+.*"
					oninput=self.link.callback(move |e: InputData|
						Msg::InputTag(id, e.value)
					)
				/>
				{
					if close_button {
						html! {
							<a
								class="act"
								onclick=self.link.callback(
									move |_| Msg::RemoveTag(id),
								)
							>
								{"X"}
							</a>
						}
					} else {
						html!{}
					}
				}
			</span>
		}
	}
}
