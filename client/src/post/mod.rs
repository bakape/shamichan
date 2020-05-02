mod countries;
pub mod image_search;
mod menu;

use super::state::{self, FeedID, Focus, Location, Post as Data, State};
use crate::buttons::SpanButton;
use crate::util;
use protocol::payloads::{FileType, Image};
use yew::{html, Component, ComponentLink, Html, NodeRef, Properties};

// Central thread container
pub struct Post {
	#[allow(unused)]
	bridge: state::HookBridge,

	#[allow(unused)]
	link: ComponentLink<Self>,

	props: Props,

	reveal_image: bool,
	expand_image: bool,
	tall_image: bool,
	image_download_button: NodeRef,
	media_el: NodeRef,
	el: NodeRef,
}

pub enum Message {
	Rerender,
	ImageHideToggle,
	ImageContract,
	ImageExpand,
	ImageDownload,
	SetVolume,
	CheckTallImage,
	NOP,
}

#[derive(Clone, Properties, PartialEq, Eq)]
pub struct Props {
	// Post ID
	pub id: u64,

	// Post rendered outside of thread content and thus should not be
	// anchor-navigable to
	#[prop_or_default]
	pub outside_thread: bool,
}

impl Component for Post {
	comp_prop_change! {Props}
	type Message = Message;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		use state::{hook, Change};

		Self {
			bridge: hook(
				&link,
				&[Change::Configs, Change::Options, Change::Post(props.id)],
				|_| Message::Rerender,
			),
			props,
			link,
			reveal_image: false,
			expand_image: false,
			tall_image: false,
			el: Default::default(),
			image_download_button: Default::default(),
			media_el: Default::default(),
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::Rerender => true,
			Message::NOP => false,
			Message::ImageHideToggle => {
				self.reveal_image = !self.reveal_image;
				true
			}
			Message::ImageExpand => {
				self.expand_image = true;
				// TODO: Hide any hover previews
				true
			}
			Message::ImageContract => {
				self.expand_image = false;
				if self.tall_image {
					// TODO: Check this does not need to be deferred to next
					// frame
					self.scroll_to();
				}
				self.tall_image = false;
				true
			}
			Message::ImageDownload => {
				if let Some(el) = self
					.image_download_button
					.cast::<web_sys::HtmlAnchorElement>()
				{
					el.click();
				}
				false
			}
			Message::SetVolume => {
				if let Some(el) =
					self.media_el.cast::<web_sys::HtmlAudioElement>()
				{
					el.set_volume(state::read(|s| {
						s.options.audio_volume as f64 / 100_f64
					}));
				}
				false
			}
			Message::CheckTallImage => {
				state::read(|s| {
					if let (Some(img), Some(wh)) = (
						s.posts
							.get(&self.props.id)
							.map(|p| p.image.as_ref())
							.flatten(),
						util::window()
							.inner_height()
							.ok()
							.map(|h| h.as_f64())
							.flatten(),
					) {
						if img.common.width as f64 > wh {
							self.scroll_to();
						}
					}
				});
				false
			}
		}
	}

	fn view(&self) -> Html {
		state::read(|s| {
			let p = match s.posts.get(&self.props.id) {
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
				<article
					id=if !self.props.outside_thread {
						format!("p-{}", self.props.id)
					} else {
						"".into()
					}
					class=cls.join(" ")
					ref=self.el.clone()
				>
					{self.render_header(s, p)}
					{
						match &p.image {
							Some(img) => self.render_figcaption(s, img),
							None => html! {},
						}
					}
					<div class="post-container">
						{
							match &p.image {
								Some(img) => self.render_figure(s, img),
								None => html! {},
							}
						}
					</div>
					// TODO: post moderation log
					// TODO: backlinks
				</article>
			}
		})
	}
}

impl Post {
	fn scroll_to(&self) {
		if let Some(el) = self.el.cast::<web_sys::Element>() {
			el.scroll_into_view();
		}
	}

	fn render_header(&self, s: &State, p: &Data) -> Html {
		let thread = if p.id == p.thread {
			s.threads.get(&p.thread)
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
				{self.render_name(s, p)}
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
					if thread.is_some()
					   && !state::read(|s| s.location.is_thread())
					{
						let id = self.props.id;
						html! {
							<>
								<SpanButton
									text="top"
									on_click=self.link.callback(move |_| {
										state::navigate_to(Location{
											feed: FeedID::Thread{
												id,
												page: 0,
											},
											focus: Some(Focus::Top),
										});
										Message::NOP
									})
								/>
								<SpanButton
									text="bottom"
									on_click=self.link.callback(move |_| {
										state::navigate_to(Location{
											feed: FeedID::Thread{
												id,
												page: -1,
											},
											focus: Some(Focus::Bottom),
										});
										Message::NOP
									})
								/>
							</>
						}
					} else {
						html! {}
					}
				}
				<menu::Menu id=self.props.id />
			</header>
		}
	}

	fn render_name(&self, s: &State, p: &Data) -> Html {
		// TODO: Staff titles

		let mut w: Vec<Html> = Default::default();

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
		if s.mine.contains(&self.props.id) {
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

	fn render_figcaption(&self, s: &State, img: &Image) -> Html {
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

		let name =
			format!("{}.{}", img.common.name, img.common.file_type.extension());

		html! {
			<figcaption class="spaced">
				{
					if s.options.hide_thumbnails || s.options.work_mode {
						html! {
							<crate::buttons::SpanButton
								text=if self.reveal_image {
									"hide"
								} else {
									"show"
								}
								on_click=self.link.callback(|_|
									Message::ImageHideToggle
								)
							/>
						}
					} else {
						html! {}
					}
				}
				{self.render_image_search(s, img)}
				<span class="file-info">
					{
						for file_info.into_iter().map(|s| html!{
							<span>{s}</span>
						})
					}
				</span>
				{
					if self.expand_image && is_expandable(img.common.file_type)
					{
						html! {
							<SpanButton
								text="contract"
								on_click=self.link.callback(|_|
									Message::ImageContract
								)
							/>
						}
					} else {
						html! {}
					}
				}
				<a
					href=source_path(s, img)
					download=name
					ref=self.image_download_button.clone()
				>
					{name}
				</a>
			</figcaption>
		}
	}

	fn render_image_search(&self, s: &State, img: &Image) -> Html {
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
			util::host(),
			root,
			hex::encode(&img.sha1),
			typ.extension(),
		);

		let mut v = Vec::<(&'static str, String)>::new();
		for p in s.options.enabled_image_search.iter() {
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

	fn render_figure(&self, s: &State, img: &Image) -> Html {
		use yew::events::MouseEvent;

		if !self.reveal_image
			&& (s.options.hide_thumbnails || s.options.work_mode)
		{
			return html! {};
		}

		let src = source_path(s, img);
		let thumb: Html;
		let is_audio = match img.common.file_type {
			FileType::MP3 | FileType::FLAC => true,
			FileType::WEBM | FileType::MP4 | FileType::OGG => !img.common.video,
			_ => false,
		};

		let (w, h, url) = if !self.expand_image || is_audio {
			if img.common.thumb_type == FileType::NoFile {
				// No thumbnail exists
				(
					150,
					150,
					match img.common.file_type {
						FileType::WEBM
						| FileType::MP4
						| FileType::MP3
						| FileType::OGG
						| FileType::FLAC => "/assets/audio.png",
						_ => "/assets/file.png",
					}
					.to_string(),
				)
			} else if img.common.spoilered && !s.options.reveal_image_spoilers {
				// Spoilered and spoilers enabled
				(150, 150, "/assets/spoil/default.jpg".into())
			} else if img.common.file_type == FileType::GIF
				&& s.options.expand_gif_thumbnails
			{
				// Animated GIF thumbnails
				(img.common.thumb_width, img.common.thumb_height, src.clone())
			} else {
				(
					img.common.thumb_width,
					img.common.thumb_height,
					thumb_path(s, img),
				)
			}
		} else {
			(img.common.width, img.common.height, src.clone())
		};

		if self.expand_image && !is_audio {
			use state::ImageExpansionMode;

			let mut cls = vec!["expanded"];
			match s.options.image_expansion_mode {
				ImageExpansionMode::FitWidth => {
					self.link.send_message(Message::CheckTallImage);
					cls.push("fit-to-width");
				}
				ImageExpansionMode::FitHeight => {
					cls.push("fit-to-height");
				}
				ImageExpansionMode::FitScreen => {
					cls.push("fit-to-width fit-to-height");
				}
				_ => (),
			};
			let cls_joined = cls.join(" ");

			let contract = self.link.callback(move |e: MouseEvent| {
				if e.button() != 0 {
					Message::NOP
				} else {
					e.prevent_default();
					Message::ImageContract
				}
			});

			thumb = match img.common.file_type {
				FileType::OGG | FileType::MP4 | FileType::WEBM => {
					self.link.send_message(Message::SetVolume);
					html! {
						<video
							ref=self.media_el.clone()
							src=url
							cls=cls_joined
							autoplay=true
							controls=true
							loop=true
							onclick=contract
						/>
					}
				}
				_ => {
					html! {
						<img
							src=url
							width=w
							height=h
							cls=cls_joined
							onclick=contract
						/>
					}
				}
			};
		} else {
			let no_mode = s.options.image_expansion_mode
				== state::ImageExpansionMode::None;
			let is_expandable = is_expandable(img.common.file_type);
			let on_click = self.link.callback(move |e: MouseEvent| {
				if no_mode || e.button() != 0 {
					Message::NOP
				} else {
					e.prevent_default();
					if is_audio || is_expandable {
						Message::ImageExpand
					} else {
						Message::ImageDownload
					}
				}
			});

			thumb = html! {
				<img
					src=url
					width=w
					height=h
					onclick=on_click
					// TODO: Image hover preview
				/>
			};
		}

		html! {
			<figure>
				{thumb}
				{
					if self.expand_image && is_audio {
						// Change volume after render
						self.link.send_message(Message::SetVolume);
						html! {
							<audio
								ref=self.media_el.clone()
								autoplay=true
								loop=true
								controls=true
								src=src,
							/>
						}
					} else {
						html! {}
					}
				}
			</figure>
		}
	}
}

// Returns root url for storing images
fn image_root(s: &State) -> &str {
	let over = &s.configs.image_root_override;
	if over.is_empty() {
		"/assets/images"
	} else {
		over
	}
}

// Get the thumbnail path of an upload
fn thumb_path(s: &State, img: &Image) -> String {
	format!(
		"{}/thumb/{}.{}",
		image_root(s),
		hex::encode(&img.sha1),
		img.common.thumb_type.extension()
	)
}

// Resolve the path to the source file of an upload
fn source_path(s: &State, img: &Image) -> String {
	format!(
		"{}/thumb/{}.{}",
		image_root(s),
		hex::encode(&img.sha1),
		img.common.file_type.extension()
	)
}

fn is_expandable(t: FileType) -> bool {
	match t {
		// Nothing to preview for these
		FileType::PDF
		| FileType::MP3
		| FileType::FLAC
		| FileType::ZIP
		| FileType::SevenZip
		| FileType::TXZ
		| FileType::TGZ
		| FileType::TXT
		| FileType::RAR
		| FileType::CBR
		| FileType::CBZ => false,
		_ => true,
	}
}
