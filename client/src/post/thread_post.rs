use super::common::{Ctx, PostCommon, PostComponent};
use yew::Html;

#[derive(Default)]
pub struct Inner {}

// A post rendered inside a a thread
pub type ThreadPost = PostCommon<Inner>;

impl PostComponent for Inner {
	type MessageExtra = ();

	fn render_body<'s, 'c>(&self, c: &Ctx<'s, 'c, Self>) -> Html {
		super::body::render(c, &c.post.body)
	}

	fn should_render(&self, props: &super::common::Props) -> bool {
		crate::state::read(|s| match s.open_post_id {
			Some(open) => open != props.id,
			None => true,
		})
	}
}
