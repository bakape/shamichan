mod countries;
pub mod image_search;
mod menu;

use super::state;
use crate::buttons::SpanButton;
use crate::util;
use protocol::{FileType, Image};
use state::Post as Data;
use yew::{
	html, Bridge, Bridged, Component, ComponentLink, Html, NodeRef, Properties,
};

// Central thread container
pub struct Post {
	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,

	#[allow(unused)]
	link: ComponentLink<Self>,

	id: u64,

	reveal_image: bool,
	expand_image: bool,
	tall_image: bool,
	image_download_button: NodeRef,
	media_el: NodeRef,
	el: NodeRef,
}

pub enum Message {
	PostChange,
	OptionsChange,
	ImageHideToggle,
	ImageContract,
	ImageExpand,
	ImageDownload,
	SetVolume,
	ScrollTo,
	CheckTallImage,
	NOP,
}

#[derive(Clone, Properties)]
pub struct Props {
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
		s.send(Request::Subscribe(Subscription::OptionsChange));
		s.send(Request::Subscribe(Subscription::ConfigsChange));

		Self {
			id: props.id,
			state: s,
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
			Message::PostChange | Message::OptionsChange => true,
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
					el.set_volume(
						state::get().options.audio_volume as f64 / 100_f64,
					);
				}
				false
			}
			Message::CheckTallImage => {
				if let (Some(img), Some(wh)) = (
					state::get()
						.posts
						.get(&self.id)
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
				false
			}
			Message::ScrollTo => {
				self.scroll_to();
				false
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
			<article
				id=format!("p-{}", self.id)
				class=cls.join(" ")
				ref=self.el.clone()
			>
				{self.render_header(p)}
				{
					match &p.image {
						Some(img) => self.render_figcaption(img),
						None => html! {},
					}
				}
				<div class="post-container">
					{
						match &p.image {
							Some(img) => self.render_figure(img),
							None => html! {},
						}
					}
				</div>
				// TODO: post moderation log
				// TODO: backlinks
			</article>
		}
	}
}

impl Post {
	fn scroll_to(&self) {
		if let Some(el) = self.el.cast::<web_sys::Element>() {
			el.scroll_into_view();
		}
	}

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

		let name =
			format!("{}.{}", img.common.name, img.common.file_type.extension());

		html! {
			<figcaption class="spaced">
				{
					if opts.hide_thumbnails || opts.work_mode {
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
				{self.render_image_search(img)}
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
					href=source_path(img)
					download=name
					ref=self.image_download_button.clone()
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
			util::host(),
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

	fn render_figure(&self, img: &Image) -> Html {
		use yew::events::MouseEvent;

		let opts = &state::get().options;
		if !self.reveal_image && (opts.hide_thumbnails || opts.work_mode) {
			return html! {};
		}

		let src = source_path(img);
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
			} else if img.common.spoilered && !opts.reveal_image_spoilers {
				// Spoilered and spoilers enabled
				(150, 150, "/assets/spoil/default.jpg".into())
			} else if img.common.file_type == FileType::GIF
				&& opts.expand_gif_thumbnails
			{
				// Animated GIF thumbnails
				(img.common.thumb_width, img.common.thumb_height, src.clone())
			} else {
				(
					img.common.thumb_width,
					img.common.thumb_height,
					thumb_path(img),
				)
			}
		} else {
			(img.common.width, img.common.height, src.clone())
		};

		if self.expand_image && !is_audio {
			use state::ImageExpansionMode;

			let mut cls = vec!["expanded"];
			match opts.image_expansion_mode {
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
			let no_mode =
				opts.image_expansion_mode == state::ImageExpansionMode::None;
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
fn image_root<'a>() -> &'a str {
	let over = &state::get().configs.image_root_override;
	if over.is_empty() {
		"/assets/images"
	} else {
		over
	}
}

// Get the thumbnail path of an upload
fn thumb_path(img: &Image) -> String {
	format!(
		"{}/thumb/{}.{}",
		image_root(),
		hex::encode(&img.sha1),
		img.common.thumb_type.extension()
	)
}

// Resolve the path to the source file of an upload
fn source_path(img: &Image) -> String {
	format!(
		"{}/thumb/{}.{}",
		image_root(),
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
