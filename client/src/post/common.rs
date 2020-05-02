use super::state;
use yew::{html, Component, ComponentLink, Html, Properties};

#[derive(Properties, PartialEq, Eq, Clone)]
pub struct Props {
	// Post ID
	pub id: u64,
}

// Context passed to PostComponent on render
pub struct RenderCtx<'s, 'c, PC>
where
	PC: PostComponent + 'static,
{
	// Global state reference
	pub app: &'s state::State,

	// Properties passed to view
	pub props: &'c Props,

	// Link to the component for yew integrations
	pub link: &'c ComponentLink<Wrapper<PC>>,

	// Post data of target post
	pub post: &'s state::Post,
}

// Helper trait for implementing components for rendering parts of a post
pub trait PostComponent: Sized + Default {
	// Message past to component on updates
	type Message;

	// Global state change events to listen to in addition to post update events
	fn listen_to() -> &'static [state::Change] {
		&[]
	}

	// Returns the value used to signal to the component a needed rerender on
	// global state change
	fn need_rerender_message() -> Self::Message;

	// Optionally run some extra init logic
	#[allow(unused_variables)]
	fn init(&mut self, link: &ComponentLink<Wrapper<Self>>) {}

	// Update a component in response to a message, returning, if it should
	// rerender
	fn update(&mut self, msg: Self::Message) -> bool;

	// Render component HTML contents
	fn view<'s, 'c>(&self, c: &RenderCtx<'s, 'c, Self>) -> Html;
}

// Wraps a PostComponent to implement yew::Component for it
pub struct Wrapper<PC>
where
	PC: PostComponent + 'static,
{
	props: Props,
	inner: PC,

	#[allow(unused)]
	bridge: state::HookBridge,
	#[allow(unused)]
	link: ComponentLink<Self>,
}

impl<PC> Component for Wrapper<PC>
where
	PC: PostComponent + 'static,
{
	comp_prop_change! {Props}
	type Message = PC::Message;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut events = vec![state::Change::Post(props.id)];
		events.extend(PC::listen_to());
		let mut s = Self {
			props,
			inner: Default::default(),
			bridge: state::hook(&link, &events, |_| {
				PC::need_rerender_message()
			}),
			link,
		};
		s.inner.init(&s.link);
		s
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		self.inner.update(msg)
	}

	fn view(&self) -> Html {
		state::read(|s| match s.posts.get(&self.props.id) {
			Some(p) => self.inner.view(&RenderCtx {
				app: s,
				props: &self.props,
				link: &self.link,
				post: p,
			}),
			None => html! {},
		})
	}
}
