mod countries;
pub mod image_search;
mod menu;

use super::state;
use crate::buttons::SpanButton;
use crate::util;
use protocol::{FileType, Image};
use state::Post as Data;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html, Properties};

// Central thread container
pub struct Post {
	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,

	#[allow(unused)]
	link: ComponentLink<Self>,

	id: u64,

	reveal_image: bool,
}

pub enum Message {
	PostChange,
	OptionsChange,
	ImageHideToggle,
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
		use state::{Agent, Request, Subscription};

		let mut s = Agent::bridge(link.callback(|u| match u {
			Subscription::PostChange(_) => Message::PostChange,
			Subscription::OptionsChange => Message::OptionsChange,
			_ => Message::NOP,
		}));
		s.send(Request::Subscribe(Subscription::PostChange(props.id)));
		Self {
			id: props.id,
			state: s,
			link,
			reveal_image: false,
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::PostChange | Message::OptionsChange => true,
			Message::NOP => false,
			Message::ImageHideToggle => {
				self.reveal_image = !self.reveal_image;
				true
			}
		}
	}

	fn view(&self) -> Html {
		let p = match state::get().posts.get(&self.id) {
			Some(p) => p,
			None => {
				return html! {};
			}
		};

		let mut cls = vec!["glass"];
		if p.open {
			cls.push("open");
		}
		if p.id == p.thread {
			cls.push("op");
		}

		html! {
			<article id={format!("p-{}", self.id)} class=cls.join(" ")>
				{self.render_header(p)}
				{
					match &p.image {
						Some(img) => self.render_figcaption(img),
						None => html! {},
					}
				}
			</article>
		}
	}
}

impl Post {
	fn render_header(&self, p: &Data) -> Html {
		let thread = if p.id == p.thread {
			state::get().threads.get(&p.thread)
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
				{self.render_name(p)}
				{
					match &p.flag {
						Some(code) => match countries::get_name(&code) {
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
				<crate::time::view::View time=p.created_on />
				<nav class="spaced">
					// TODO: focus this post
					<a>{"#"}</a>
					// TODO: quote this post
					<a>{p.id}</a>
				</nav>
				{
					if thread.is_some() {
						html! {
							// TODO
							<>
								<SpanButton
									text="top"
									on_click=self.link.callback(|_|
										Message::NOP
									)
								/>
								<SpanButton
									text="bottom"
									on_click=self.link.callback(|_|
										Message::NOP
									)
								/>
							</>
						}
					} else {
						html! {}
					}
				}
				<menu::Menu id=self.id />
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

	fn render_figcaption(&self, img: &Image) -> Html {
		let opts = &state::get().options;
		let mut file_info = Vec::<String>::new();

		#[rustfmt::skip]
		macro_rules! push_if {
			($cond:expr, $value:expr) => {
				if $cond {
					file_info.push($value);
				}
			};
		}

		push_if!(img.common.audio, "♫".into());
		push_if!(
			img.common.duration != 0,
			util::format_duration(img.common.duration)
		);
		file_info.push({
			let s = img.common.size;
			if s < 1 << 10 {
				format!("{} B", s)
			} else if s < 1 << 20 {
				format!("{} KB", s / (1 << 20))
			} else {
				format!("{:.1} MB", s as f32 / (1 << 20) as f32)
			}
		});
		push_if!(
			img.common.width != 0 || img.common.height != 0,
			format!("{}x{}", img.common.width, img.common.height)
		);

		if let Some(a) = &img.common.artist {
			file_info.push(a.clone());
			if img.common.title.is_some() {
				file_info.push(" - ".into());
			}
		}
		if let Some(t) = &img.common.title {
			file_info.push(t.clone());
		}

		let ext = img.common.file_type.extension();
		let name = format!("{}.{}", img.common.name, ext);

		html! {
			<figcaption class="spaced">
				{
					if opts.hide_thumbnails || opts.work_mode {
						html! {
							<crate::buttons::SpanButton
								text=localize!(
									if self.reveal_image {
										"hide"
									} else {
										"show"
									}
								)
								on_click=self.link.callback(|_|
									Message::ImageHideToggle
								)
							/>
						}
					} else {
						html! {}
					}
				}
				{self.render_image_search(img)}
				<span class="file-info">
					{
						for file_info.into_iter().map(|s| html!{
							<span>{s}</span>
						})
					}
				</span>
				<a
					href=format!(
						"/assets/images/src/{}.{}",
						hex::encode(&img.sha1),
						ext
					)
					download=name
				>
					{name}
				</a>
			</figcaption>
		}
	}

	fn render_image_search(&self, img: &Image) -> Html {
		match img.common.thumb_type {
			FileType::NoFile | FileType::PDF => return html! {},
			_ => (),
		};

		// Resolve URL of image search providers, that require to download the
		// image file.
		//
		// 8 MB is the size limit on many engines.
		let (root, typ) =
			match (&img.common.file_type, img.common.size < 8 << 20) {
				(FileType::JPEG, true)
				| (FileType::PNG, true)
				| (FileType::GIF, true) => ("src", &img.common.file_type),
				_ => ("thumb", &img.common.thumb_type),
			};
		let url = format!(
			"{}/assets/images/{}/{}.{}",
			util::window().location().host().unwrap(),
			root,
			hex::encode(&img.sha1),
			typ.extension(),
		);

		let mut v = Vec::<(&'static str, String)>::new();
		for p in state::get().options.enabled_image_search.iter() {
			if let Some(u) = p.url(img, &url) {
				v.push((p.symbol(), u));
			}
		}

		html! {
			<span class="spaced">
				{
					for v.into_iter().map(|(s, u)| {
						html! {
							<a
								class="image-search"
								target="_blank"
								rel="nofollow"
								href=u
							>
								{s}
							</a>
						}
					})
				}
			</span>
		}
	}
}
