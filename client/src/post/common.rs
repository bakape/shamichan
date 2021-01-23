use crate::{
	buttons::SpanButton,
	comp_util,
	mouse::Coordinates,
	state::{self, FeedID, Focus, Location},
	util,
};
use common::{
	debug_log,
	payloads::{FileType, Image, Post},
};
use yew::{html, ComponentLink, Html, NodeRef, Properties};

#[derive(Clone, Properties, PartialEq, Eq, Debug)]
pub struct Props {
	/// Post ID
	pub id: u64,
}

pub trait PostComponent: Default {
	/// Message used to handle additional logic
	type MessageExtra;

	/// Extra initialization logic
	#[allow(unused_variables)]
	fn init(&mut self, link: &ComponentLink<PostCommon<Self>>) {}

	/// MessageExtra handler. Returns, if component should be rerendered
	#[allow(unused_variables)]
	fn update_extra<'s, 'c>(
		&mut self,
		ctx: &mut CtxMut<'s, 'c, Self>,
		msg: Self::MessageExtra,
	) -> bool {
		false
	}

	/// Should post even render?
	#[allow(unused_variables)]
	fn should_render(&self, props: &Props) -> bool {
		true
	}

	/// Return, if this component is a preview of a post and thus should not
	/// spawn its own previews.
	//
	/// Value must be static.
	fn is_preview() -> bool {
		false
	}

	/// Can be dragged and repositioned across the screen
	fn is_draggable(props: &Props) -> bool;

	/// Extra classes to assign to the post's root element
	fn extra_classes(&self) -> &'static [&'static str] {
		Default::default()
	}

	/// Render post's text body
	fn render_body<'s, 'c>(&self, c: &Ctx<'s, 'c, Self>) -> Html;

	/// Append extra HTML to the end of the post's root element
	#[allow(unused_variables)]
	fn render_after<'s, 'c>(&self, c: &Ctx<'s, 'c, Self>) -> Html {
		html! {}
	}
}

/// Context passed to PostComponent implementors
pub struct Ctx<'s, 'c, PC>
where
	PC: PostComponent + 'static,
{
	/// Global state reference
	pub app: &'s state::State,

	/// Post data of target post
	pub post: &'s Post,

	/// comp_util::Ctx passed from upstream
	pub ctx: &'c comp_util::Ctx<PostCommonInner<PC>>,
}

/// Partially mutable context passed to PostComponent implementors
pub struct CtxMut<'s, 'c, PC>
where
	PC: PostComponent + 'static,
{
	/// Global state reference
	pub app: &'s state::State,

	/// comp_util::Ctx passed from upstream
	pub ctx: &'c mut comp_util::Ctx<PostCommonInner<PC>>,
}

/// Common behavior for all post PostComponents as a wrapper
pub type PostCommon<PC> = comp_util::HookedComponent<PostCommonInner<PC>>;

/// Implements comp_util::Inner for PostCommon
#[derive(Default)]
pub struct PostCommonInner<PC>
where
	PC: PostComponent + 'static,
{
	inner: PC,

	reveal_image: bool,
	expand_image: bool,
	tall_image: bool,

	/// None, if not currently dragging
	drag_agent: Option<Box<dyn yew::agent::Bridge<crate::mouse::Agent>>>,
	last_mouse_coordinates: Coordinates,
	translation: Coordinates,

	image_download_button: NodeRef,
	media_el: NodeRef,
	el: NodeRef,
}

pub enum Message<E> {
	Rerender,
	ImageHideToggle,
	ImageContract,
	ImageExpand,
	ImageDownload,
	SetVolume,
	CheckTallImage,
	DragStart(Coordinates),
	MouseMove(Coordinates),
	QuoteSelf,
	NOP,
	Extra(E),
}

impl<PC> comp_util::Inner for PostCommonInner<PC>
where
	PC: PostComponent + 'static,
{
	type Properties = Props;
	type Message = Message<PC::MessageExtra>;

	fn init(&mut self, c: &mut comp_util::Ctx<Self>) {
		self.inner.init(&c.link);
	}

	fn update_message() -> Self::Message {
		Message::Rerender
	}

	fn subscribe_to(props: &Self::Properties) -> Vec<state::Change> {
		use state::Change;

		vec![Change::Configs, Change::Options, Change::Post(props.id)]
	}

	fn update(
		&mut self,
		c: &mut comp_util::Ctx<Self>,
		msg: Self::Message,
	) -> bool {
		use Message::*;

		match msg {
			Rerender => true,
			NOP => false,
			Extra(e) => state::read(|s| {
				self.inner.update_extra(&mut CtxMut { app: s, ctx: c }, e)
			}),
			ImageHideToggle => {
				self.reveal_image = !self.reveal_image;
				true
			}
			ImageExpand => {
				self.expand_image = true;
				// TODO: Hide any hover previews
				true
			}
			ImageContract => {
				self.expand_image = false;
				if self.tall_image {
					// TODO: Check, if this does not need to be deferred to next
					// AF
					self.scroll_to();
				}
				self.tall_image = false;
				true
			}
			ImageDownload => {
				if let Some(el) = self
					.image_download_button
					.cast::<web_sys::HtmlAnchorElement>()
				{
					el.click();
				}
				false
			}
			SetVolume => {
				if let Some(el) =
					self.media_el.cast::<web_sys::HtmlAudioElement>()
				{
					el.set_volume(state::read(|s| {
						s.options.audio_volume as f64 / 100_f64
					}));
				}
				false
			}
			CheckTallImage => {
				util::with_logging(|| {
					state::read(|s| {
						if let (Some(img), Some(wh)) = (
							s.posts
								.get(&c.props().id)
								.map(|p| p.image.as_ref())
								.flatten(),
							util::window().inner_height()?.as_f64(),
						) {
							if img.width as f64 > wh {
								self.scroll_to();
							}
						}
						Ok(())
					})
				});
				false
			}
			DragStart(coords) => {
				use crate::mouse;
				use yew::agent::Bridged;

				if !PC::is_draggable(c.props()) {
					return false;
				}

				self.last_mouse_coordinates = coords;
				let mut b =
					mouse::Agent::bridge(c.link.callback(|msg| match msg {
						mouse::Response::Coordinates(c) => {
							Message::MouseMove(c)
						}
						_ => Message::NOP,
					}));
				b.send(mouse::Request::StartDragging);
				self.drag_agent = Some(b);

				true
			}
			QuoteSelf => {
				use super::posting::{Agent, Request};
				use yew::agent::Dispatched;

				if let Some(el) = self.el.cast::<web_sys::Node>() {
					Agent::dispatcher().send(Request::QuotePost {
						post: c.props().id,
						target_post: el,
					});
				}
				false
			}
			MouseMove(coords) => {
				if self.drag_agent.is_none() || !PC::is_draggable(c.props()) {
					return false;
				}

				self.translation += coords - self.last_mouse_coordinates;
				self.last_mouse_coordinates = coords;
				true
			}
		}
	}

	fn view(&self, c: &comp_util::Ctx<Self>) -> Html {
		if !self.inner.should_render(c.props()) {
			debug_log!("post specified to not render", c.props().id);
			return html! {};
		}

		state::read(|s| {
			let c = Ctx {
				app: s,
				ctx: c,
				post: match s.posts.get(&c.props().id) {
					Some(p) => p,
					None => {
						debug_log!(
							"post not found in collection",
							c.props().id
						);
						return html! {};
					}
				},
			};

			let mut cls = vec!["glass"];
			cls.extend(self.inner.extra_classes());
			if c.post.open {
				cls.push("open");
			}

			let mut style = String::new();
			if !self.translation.is_zero() {
				style = format!(
					"transform: translate({}px, {}px);",
					self.translation.x, self.translation.y
				);
				cls.push("translated");
			} else if c.post.id == c.post.thread {
				// Moved OPs need to not blend into the background
				cls.push("no-border");
			}

			#[rustfmt::skip]
			macro_rules! with_image {
				($method:ident) => {
					match &c.post.image {
						Some(img) => self.$method(&c, img),
						None => html! {},
					}
				};
			}

			html! {
				<article
					class=cls
					key=c.ctx.props().id
					ref=self.el.clone()
					style=style
				>
					{self.render_header(&c)}
					{with_image!(render_figcaption)}
					<div class="post-container">
						{with_image!(render_figure)}
						<blockquote>{self.inner.render_body(&c)}</blockquote>
					</div>
					// TODO: post moderation log
					// TODO: backlinks
					{self.inner.render_after(&c)}
				</article>
			}
		})
	}
}

impl<PC> PostCommonInner<PC>
where
	PC: PostComponent + 'static,
{
	fn scroll_to(&self) {
		if let Some(el) = self.el.cast::<web_sys::Element>() {
			el.scroll_into_view();
		}
	}

	fn render_header<'s, 'c>(&self, c: &Ctx<'s, 'c, PC>) -> Html {
		let thread = if c.post.id == c.post.thread {
			c.app.threads.get(&c.post.thread)
		} else {
			None
		};

		let inner = html! {
			<>
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
					match &c.post.opts.post_opts.flag {
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
					<a
						class="quote"
						onclick=c.ctx.link.callback(|_| Message::QuoteSelf)
					>
						{c.post.id}
					</a>
				</nav>
				{
					if thread.is_some()
					&& !PC::is_preview()
					&& !state::read(|s| c.app.location.is_thread())
					{
						let id = c.post.id;
						html! {
							<>
								<SpanButton
									text="top"
									on_click=c.ctx.link.callback(move |_| {
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
									on_click=c.ctx.link.callback(move |_| {
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
				<super::menu::Menu id=c.post.id />
			</>
		};

		// TODO: return to original position on double click
		if PC::is_draggable(c.ctx.props()) {
			html! {
				<header
					class="spaced draggable"
					ondragstart=c.ctx.link.callback(|e: web_sys::DragEvent| {
						e.prevent_default();
						Message::DragStart(Coordinates::from(&*e))
					})
					draggable="true"
				>
					{inner}
				</header>
			}
		} else {
			html! {
				<header class="spaced">
					{inner}
				</header>
			}
		}
	}

	fn render_name<'s, 'c>(&self, c: &Ctx<'s, 'c, PC>) -> Html {
		// TODO: Staff titles
		let mut w: Vec<Html> = Default::default();
		let p_opts = &c.post.opts.post_opts;

		if c.app.options.forced_anonymity
			|| (p_opts.name.is_none() && p_opts.trip.is_none())
		{
			w.push(html! {
				<span>{localize!("anon")}</span>
			});
		} else {
			if let Some(name) = &p_opts.name {
				w.push(html! {
					<span>{name}</span>
				});
			}
			if let Some(trip) = &p_opts.trip {
				w.push(html! {
					<code>{trip}</code>
				});
			}
		}
		if c.app.mine.contains(&c.post.id) {
			w.push(html! {
				<i>{localize!("you")}</i>
			});
		}

		let mut cls = vec!["name"];
		if c.post.opts.sage {
			cls.push("sage");
		}
		// TODO: Add admin class, if staff title

		html! {
			<b class=cls>
				{w.into_iter().collect::<Html>()}
			</b>
		}
	}

	fn render_figcaption<'s, 'c>(
		&self,
		c: &Ctx<'s, 'c, PC>,
		img: &Image,
	) -> Html {
		let mut file_info = Vec::<String>::new();

		#[rustfmt::skip]
		macro_rules! push_if {
			($cond:expr, $value:expr) => {
				if $cond {
					file_info.push($value);
				}
			};
		}

		push_if!(img.audio, "♫".into());
		push_if!(img.duration != 0, util::format_duration(img.duration));
		file_info.push({
			let s = img.size;
			if s < 1 << 10 {
				format!("{} B", s)
			} else if s < 1 << 20 {
				format!("{} KB", s / (1 << 20))
			} else {
				format!("{:.1} MB", s as f32 / (1 << 20) as f32)
			}
		});
		push_if!(
			img.width != 0 || img.height != 0,
			format!("{}x{}", img.width, img.height)
		);

		if let Some(a) = &img.artist {
			file_info.push(a.clone());
			if img.title.is_some() {
				file_info.push(" - ".into());
			}
		}
		if let Some(t) = &img.title {
			file_info.push(t.clone());
		}

		let name = format!("{}.{}", img.name, img.file_type.extension());

		html! {
			<figcaption class="spaced">
				{
					if c.app.options.hide_thumbnails || c.app.options.work_mode
					{
						html! {
							<crate::buttons::SpanButton
								text=if self.reveal_image {
									"hide"
								} else {
									"show"
								}
								on_click=c.ctx.link.callback(|_|
									Message::ImageHideToggle
								)
							/>
						}
					} else {
						html! {}
					}
				}
				{self.render_image_search(c, img)}
				<span class="file-info">
					{
						for file_info.into_iter().map(|s| html!{
							<span>{s}</span>
						})
					}
				</span>
				{
					if self.expand_image && is_expandable(img.file_type)
					{
						html! {
							<SpanButton
								text="contract"
								on_click=c.ctx.link.callback(|_|
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

	fn render_image_search<'s, 'c>(
		&self,
		c: &Ctx<'s, 'c, PC>,
		img: &Image,
	) -> Html {
		use FileType::*;

		match img.thumb_type {
			NoFile | PDF => return html! {},
			_ => (),
		};

		// Resolve URL of image search providers, that require to download the
		// image file.
		//
		// 8 MB is the size limit on many engines.
		let (root, typ) = match (&img.file_type, img.size < 8 << 20) {
			(JPEG, true) | (PNG, true) | (GIF, true) => ("src", &img.file_type),
			_ => ("thumb", &img.thumb_type),
		};
		let url = format!(
			"{}/assets/images/{}/{}.{}",
			util::host(),
			root,
			hex::encode(&img.sha1),
			typ.extension(),
		);

		let mut v = Vec::<(&'static str, String)>::new();
		for p in c.app.options.enabled_image_search.iter() {
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

	fn render_figure<'s, 'c>(&self, c: &Ctx<'s, 'c, PC>, img: &Image) -> Html {
		use yew::events::MouseEvent;
		use FileType::*;

		if !self.reveal_image
			&& (c.app.options.hide_thumbnails || c.app.options.work_mode)
		{
			return html! {};
		}

		let src = source_path(img);
		let thumb: Html;
		let is_audio = match img.file_type {
			MP3 | FLAC => true,
			WEBM | MP4 | OGG => !img.video,
			_ => false,
		};

		let (w, h, url) = if !self.expand_image || is_audio {
			if img.thumb_type == NoFile {
				// No thumbnail exists
				(
					150,
					150,
					match img.file_type {
						WEBM | MP4 | MP3 | OGG | FLAC => "/assets/audio.png",
						_ => "/assets/file.png",
					}
					.to_string(),
				)
			} else if img.spoilered && !c.app.options.reveal_image_spoilers {
				// Spoilered and spoilers enabled
				(150, 150, "/assets/spoil/default.jpg".into())
			} else if img.file_type == GIF
				&& c.app.options.expand_gif_thumbnails
			{
				// Animated GIF thumbnails
				(img.thumb_width, img.thumb_height, src.clone())
			} else {
				(img.thumb_width, img.thumb_height, thumb_path(img))
			}
		} else {
			(img.width, img.height, src.clone())
		};

		if self.expand_image && !is_audio {
			use state::ImageExpansionMode;

			let mut cls = vec!["expanded"];
			match c.app.options.image_expansion_mode {
				ImageExpansionMode::FitWidth => {
					c.ctx.link.send_message(Message::CheckTallImage);
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

			let contract = c.ctx.link.callback(move |e: MouseEvent| {
				if e.button() != 0 {
					Message::NOP
				} else {
					e.prevent_default();
					Message::ImageContract
				}
			});

			thumb = match img.file_type {
				OGG | MP4 | WEBM => {
					c.ctx.link.send_message(Message::SetVolume);
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
			let no_mode = c.app.options.image_expansion_mode
				== state::ImageExpansionMode::None;
			let is_expandable = is_expandable(img.file_type);
			let on_click = c.ctx.link.callback(move |e: MouseEvent| {
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
						c.ctx.link.send_message(Message::SetVolume);
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

/// Get the thumbnail path of an upload
fn thumb_path(img: &Image) -> String {
	format!(
		"/assets/images/thumb/{}.{}",
		hex::encode(&img.sha1),
		img.thumb_type.extension()
	)
}

/// Resolve the path to the source file of an upload
fn source_path(img: &Image) -> String {
	format!(
		"/assets/images/thumb/{}.{}",
		hex::encode(&img.sha1),
		img.file_type.extension()
	)
}

fn is_expandable(t: FileType) -> bool {
	use FileType::*;

	match t {
		// Nothing to preview for these
		PDF | MP3 | FLAC | ZIP | SevenZip | TXZ | TGZ | TXT | RAR | CBR
		| CBZ => false,
		_ => true,
	}
}
