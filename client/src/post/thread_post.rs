use super::common::{Ctx, PostCommon, PostComponent};
use yew::Html;

#[derive(Default)]
pub struct Inner {}

/// A post rendered inside a a thread
pub type ThreadPost = PostCommon<Inner>;

impl PostComponent for Inner {
	type MessageExtra = ();

	fn render_body<'c>(&self, c: &Ctx<'c, Self>) -> Html {
		super::body::render(c, &c.post().body)
	}

	#[inline]
	fn should_render<'c>(&self, c: &Ctx<'c, Self>) -> bool {
		match c.app_state().open_post_id {
			Some(open) => open != c.props().id,
			None => true,
		}
	}

	#[inline]
	fn is_draggable<'c>(&self, c: &Ctx<'c, Self>) -> bool {
		// TODO: inlined posts should never be draggable
		c.app_state().location.is_thread()
	}
}
