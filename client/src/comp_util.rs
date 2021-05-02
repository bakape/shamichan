/// Helpers for more easily implementing components
use std::cell::RefCell;

/// Generate partial Component implementation for self.props as Self::Properties
/// updates
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

/// Generate partial Component implementation for components with no possible
/// property changes
#[macro_export]
macro_rules! comp_no_prop_change {
	($props:ty) => {
		type Properties = $props;

		fn change(&mut self, _: Self::Properties) -> bool {
			false
		}
	};
}

/// Generate partial Component implementation with no properties
#[macro_export]
macro_rules! comp_no_props {
	() => {
		type Properties = ();

		fn change(&mut self, _: Self::Properties) -> bool {
			false
		}
	};
}

/// Generate partial Component implementation for components with only a
/// rerender message
#[macro_export]
macro_rules! comp_message_rerender {
	() => {
		type Message = ();

		fn update(&mut self, _: Self::Message) -> bool {
			true
		}
	};
}

/// Generate Component update methods for components with no update messages
#[macro_export]
macro_rules! comp_no_update {
	() => {
		type Message = ();

		fn update(&mut self, _: Self::Message) -> bool {
			false
		}
	};
}

/// Generate Component update methods for static components
#[macro_export]
macro_rules! comp_static {
	($props:ty) => {
		$crate::comp_no_prop_change! {$props}
		$crate::comp_no_update! {}
	};
	() => {
		$crate::comp_static! {()}
	};
}

/// Parameters passed to Inner method calls
pub struct Ctx<I>
where
	I: Inner + 'static,
{
	props: I::Properties,
	link: yew::ComponentLink<HookedComponent<I>>,
	app_state: crate::state::StateBridge,
}

impl<I> Ctx<I>
where
	I: Inner + 'static,
{
	/// Get reference to component's properties
	#[inline]
	pub fn props(&self) -> &I::Properties {
		&self.props
	}

	/// Set component properties. Returns, if properties where altered.
	pub fn set_props(&mut self, props: I::Properties) -> bool {
		if self.props != props {
			let old = I::subscribe_to(&self.props);
			let new = I::subscribe_to(&props);
			if old != new {
				self.app_state.send(
					crate::state::Request::ChangeNotifications {
						remove: old,
						add: new,
					},
				);
			}
			self.props = props;
			true
		} else {
			false
		}
	}

	/// Get reference to component's properties
	#[inline]
	pub fn link(&self) -> &yew::ComponentLink<HookedComponent<I>> {
		&self.link
	}

	/// Get reference to the global application state.
	#[inline]
	pub fn app_state(&self) -> std::cell::Ref<'static, crate::state::State> {
		self.app_state.get()
	}
}

/// Inner logic for HookedComponent
pub trait Inner: Default {
	type Properties: yew::Properties + Eq + std::fmt::Debug;
	type Message: 'static;

	/// Extra initialization logic
	#[allow(unused_variables)]
	#[inline]
	fn init(&mut self, c: &mut Ctx<Self>) {}

	/// Return Self::Message to pass to HookedInner to signal global state has
	/// updated
	fn update_message() -> Self::Message;

	/// Vector of global state changes to subscribe to
	#[allow(unused_variables)]
	#[inline]
	fn subscribe_to(props: &Self::Properties) -> Vec<crate::state::Change> {
		Default::default()
	}

	/// Same as for yew::Component
	fn update(&mut self, c: &mut Ctx<Self>, msg: Self::Message) -> bool;

	/// Same as for yew::Component
	fn view(&self, c: &Ctx<Self>) -> yew::Html;

	/// Called each time after the component is rendered
	#[allow(unused_variables)]
	fn rendered(&mut self, c: &mut Ctx<Self>, first_render: bool) {}
}

/// Component that is hooked into global state updates
pub struct HookedComponent<I>
where
	I: Inner + 'static,
{
	inner: RefCell<I>,
	ctx: Ctx<I>,
}

impl<I> yew::Component for HookedComponent<I>
where
	I: Inner + 'static,
{
	type Properties = I::Properties;
	type Message = I::Message;

	fn create(props: Self::Properties, link: yew::ComponentLink<Self>) -> Self {
		let mut inner: I = Default::default();
		let mut ctx = Ctx {
			app_state: crate::state::hook(
				&link,
				I::subscribe_to(&props),
				|| I::update_message(),
			),
			link,
			props,
		};
		inner.init(&mut ctx);
		Self {
			inner: inner.into(),
			ctx,
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		self.inner.borrow_mut().update(&mut self.ctx, msg)
	}

	fn change(&mut self, props: Self::Properties) -> bool {
		self.ctx.set_props(props)
	}

	fn view(&self) -> yew::Html {
		self.inner.borrow().view(&self.ctx)
	}

	fn rendered(&mut self, first_render: bool) {
		self.inner
			.borrow_mut()
			.rendered(&mut self.ctx, first_render);
	}
}
