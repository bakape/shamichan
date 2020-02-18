use super::state;
use state::Post as Data;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html, Properties};

// Central thread container
pub struct Post {
	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,

	#[allow(unused)]
	link: ComponentLink<Self>,

	id: u64,
}

pub enum Message {
	PostChange,
	NOP,
}

#[derive(Clone, Properties)]
pub struct Props {
	#[props(required)]
	pub id: u64,
}

impl Component for Post {
	type Message = Message;
	type Properties = Props;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut s = state::Agent::bridge(link.callback(|u| match u {
			state::Subscription::PostChange(_) => Message::PostChange,
			_ => Message::NOP,
		}));
		s.send(state::Request::Subscribe(state::Subscription::PostChange(
			props.id,
		)));
		Self {
			id: props.id,
			state: s,
			link,
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::PostChange => true,
			Message::NOP => false,
		}
	}

	fn view(&self) -> Html {
		let p = match state::get().posts.get(&self.id) {
			Some(p) => p,
			None => {
				return html! {};
			}
		};

		html! {
			<>
				<article id={format!("p-{}", self.id)}>
					{self.render_header(p)}
				</article>
			</>
		}
	}
}

impl Post {
	fn render_header(&self, p: &Data) -> Html {
		let s = state::get();
		html! {
			<header class="spaced">
				{
					match (p.id == p.thread, s.threads.get(&p.thread)) {
						(true, Some(t)) => {
							html! {
								<>
									{
										for t.tags.iter().map(|t| {
											html! {
												<b>{format!("/{}/", t)}</b>
											}
										})
									}
									<h3>{format!("「{}」", t.subject)}</h3>
								</>
							}
						},
						_ => html! {}
					}
				}
				{self.render_name(p)}
				<crate::time::view::View time=p.created_on />
			</header>
		}
	}

	fn render_name(&self, p: &Data) -> Html {
		// TODO: Staff titles

		let mut w: Vec<Html> = Default::default();
		let s = state::get();

		if s.options.forced_anonymity || (p.name.is_none() && p.trip.is_none())
		{
			w.push(html! {
				<span>{localize!("anon")}</span>
			});
		} else {
			if let Some(name) = &p.name {
				w.push(html! {
					<span>{name}</span>
				});
			}
			if let Some(trip) = &p.trip {
				w.push(html! {
					<code>{trip}</code>
				});
			}
		}
		if s.mine.contains(&self.id) {
			w.push(html! {
				<i>{localize!("you")}</i>
			});
		}

		html! {
			<b class="name">
				{w.into_iter().collect::<Html>()}
			</b>
		}
	}
}
