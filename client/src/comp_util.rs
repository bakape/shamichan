// Helpers for more easily implementing components

use std::cell::RefCell;

// Generate partial Component implementation for self.props as Self::Properties
// updates
#[macro_export]
macro_rules! comp_prop_change {
	($props:ty) => {
		type Properties = $props;

		fn change(&mut self, props: Self::Properties) -> bool {
			if self.props != props {
				self.props = props;
				true
			} else {
				false
			}
		}
	};
}

// Generate partial Component implementation for components with no possible
// property changes
#[macro_export]
macro_rules! comp_no_prop_change {
	($props:ty) => {
		type Properties = $props;

		fn change(&mut self, _: Self::Properties) -> bool {
			false
		}
	};
}

// Generate partial Component implementation with no properties
#[macro_export]
macro_rules! comp_no_props {
	() => {
		type Properties = ();

		fn change(&mut self, _: Self::Properties) -> bool {
			false
		}
	};
}

// Generate partial Component implementation for components with only a
// rerender message
#[macro_export]
macro_rules! comp_message_rerender {
	() => {
		type Message = ();

		fn update(&mut self, _: Self::Message) -> bool {
			true
		}
	};
}

// Generate Component update methods for static components
#[macro_export]
macro_rules! comp_static {
	($props:ty) => {
		$crate::comp_no_prop_change! {$props}
		type Message = ();

		fn update(&mut self, _: Self::Message) -> bool {
			false
		}
	};
	() => {
		$crate::comp_static! {()}
	};
}

// Parameters passed to HookedInner method calls
pub struct Ctx<'a, I>
where
	I: Inner + 'static,
{
	pub props: &'a I::Properties,
	pub link: &'a yew::ComponentLink<HookedComponent<I>>,
	pub bridge: &'a crate::state::HookBridge,
}

// Inner logic for HookedComponent
pub trait Inner: Default {
	type Properties: yew::Properties + Eq;
	type Message: 'static;

	// Extra initialization logic
	#[allow(unused_variables)]
	fn init<'a>(&mut self, c: Ctx<'a, Self>) {}

	// Return Self::Message to pass to HookedInner to signal global state has
	// updated
	fn update_message() -> Self::Message;

	// Vector of global state changes to subscribe to
	#[allow(unused_variables)]
	fn subscribe_to(props: &Self::Properties) -> Vec<crate::state::Change> {
		Default::default()
	}

	// Same as for yew::Component
	fn update<'a>(&mut self, c: Ctx<'a, Self>, msg: Self::Message) -> bool;

	// Same as for yew::Component
	fn view<'a>(&self, c: Ctx<'a, Self>) -> yew::Html;
}

// Component that is hooked into global state updates
pub struct HookedComponent<I>
where
	I: Inner + 'static,
{
	inner: RefCell<I>,
	props: I::Properties,

	#[allow(unused)]
	bridge: crate::state::HookBridge,
	#[allow(unused)]
	link: yew::ComponentLink<Self>,
}

impl<I> yew::Component for HookedComponent<I>
where
	I: Inner + 'static,
{
	type Properties = I::Properties;
	type Message = I::Message;

	fn create(props: Self::Properties, link: yew::ComponentLink<Self>) -> Self {
		let s = Self {
			inner: Default::default(),
			bridge: crate::state::hook(&link, I::subscribe_to(&props), |_| {
				I::update_message()
			}),
			link,
			props,
		};
		s.inner.borrow_mut().init(s.ctx());
		s
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		self.inner.borrow_mut().update(self.ctx(), msg)
	}

	fn change(&mut self, props: Self::Properties) -> bool {
		if self.props != props {
			let old = I::subscribe_to(&self.props);
			let new = I::subscribe_to(&props);
			if old != new {
				self.bridge
					.send(crate::state::Request::ChangeNotifications {
						remove: old,
						add: new,
					});
			}
			self.props = props;
			true
		} else {
			false
		}
	}

	fn view(&self) -> yew::Html {
		self.inner.borrow().view(self.ctx())
	}
}

impl<I> HookedComponent<I>
where
	I: Inner + 'static,
{
	// Return context to pass to Inner method calls
	fn ctx(&self) -> Ctx<'_, I> {
		Ctx {
			props: &self.props,
			link: &self.link,
			bridge: &self.bridge,
		}
	}
}
