use super::common::{PostCommon, PostComponent, RenderCtx};
use yew::Html;

#[derive(Default)]
pub struct Inner {}

// A post rendered inside a a thread
pub type ThreadPost = PostCommon<Inner>;

impl PostComponent for Inner {
	type MessageExtra = ();

	fn render_id<'s, 'c>(&self, c: &RenderCtx<'s, 'c, Self>) -> String {
		format!("p-{}", c.post.id)
	}

	fn render_body<'s, 'c>(&self, c: &RenderCtx<'s, 'c, Self>) -> Html {
		super::body::render(c, &c.post.body)
	}
}
