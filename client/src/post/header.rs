use super::common::{PostComponent, RenderCtx, Wrapper};
use crate::state;
use yew::{html, Html};

pub type Header = Wrapper<Inner>;

#[derive(Default)]
pub struct Inner {}

impl PostComponent for Inner {
	type Message = bool;

	fn listen_to() -> &'static [state::Change] {
		&[state::Change::Options]
	}

	fn need_rerender_message() -> Self::Message {
		true
	}

	fn update(&mut self, _: Self::Message) -> bool {
		false
	}

	fn view<'s, 'c>(&self, c: &RenderCtx<'s, 'c, Self>) -> Html {
		use crate::buttons::SpanButton;
		use crate::state::{FeedID, Focus, Location};

		let thread = if c.post.id == c.post.thread {
			c.app.threads.get(&c.post.thread)
		} else {
			None
		};

		html! {
			<header class="spaced">
				{
					match thread {
						Some(t) => {
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
				{self.render_name(c)}
				{
					match &c.post.flag {
						Some(code) => match super::countries::get_name(&code) {
							Some(name) => html! {
								<img
									class="flag"
									src=format!("/assets/flags/{}.svg", &code)
									title=name
								/>
							},
							None => html! {},
						}
						None => html! {},
					}
				}
				<crate::time::view::View time=c.post.created_on />
				<nav class="spaced">
					// TODO: focus this post
					<a>{"#"}</a>
					// TODO: quote this post
					<a>{c.post.id}</a>
				</nav>
				{
					if thread.is_some()
					   && !state::read(|s| s.location.is_thread())
					{
						let id = c.props.id;
						html! {
							<>
								<SpanButton
									text="top"
									on_click=c.link.callback(move |_| {
										state::navigate_to(Location{
											feed: FeedID::Thread{
												id,
												page: 0,
											},
											focus: Some(Focus::Top),
										});
										false
									})
								/>
								<SpanButton
									text="bottom"
									on_click=c.link.callback(move |_| {
										state::navigate_to(Location{
											feed: FeedID::Thread{
												id,
												page: -1,
											},
											focus: Some(Focus::Bottom),
										});
										false
									})
								/>
							</>
						}
					} else {
						html! {}
					}
				}
				<super::menu::Menu id=c.props.id />
			</header>
		}
	}
}

impl Inner {
	fn render_name<'s, 'c>(&self, c: &RenderCtx<'s, 'c, Self>) -> Html {
		// TODO: Staff titles

		let mut w: Vec<Html> = Default::default();
		let p = c.post;

		if c.app.options.forced_anonymity
			|| (p.name.is_none() && p.trip.is_none())
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
		if c.app.mine.contains(&c.props.id) {
			w.push(html! {
				<i>{localize!("you")}</i>
			});
		}

		let mut cls = vec!["name"];
		if p.sage {
			cls.push("sage");
		}
		// TODO: Add admin class, if staff title

		html! {
			<b class=cls.join(" ")>
				{w.into_iter().collect::<Html>()}
			</b>
		}
	}
}
