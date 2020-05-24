use super::{buttons, comp_util, state};
use yew::{html, Html, Properties};

// Central thread container
pub type Thread = comp_util::HookedComponent<Inner>;

#[derive(Default)]
pub struct Inner {}

// Posts to display in a thread
#[derive(Clone, Eq, PartialEq)]
pub enum PostSet {
	// Display OP + last 5 posts
	Last5Posts,

	// Display OP + selected page
	Page(u32),
}

impl Default for PostSet {
	fn default() -> Self {
		Self::Last5Posts
	}
}

#[derive(Clone, Properties, Eq, PartialEq)]
pub struct Props {
	pub id: u64,
	pub pages: PostSet,
}

impl comp_util::Inner for Inner {
	type Message = ();
	type Properties = Props;

	fn update_message() -> Self::Message {
		()
	}

	fn subscribe_to(props: &Self::Properties) -> Vec<state::Change> {
		vec![state::Change::Thread(props.id)]
	}

	fn update<'a>(
		&mut self,
		_: comp_util::Ctx<'a, Self>,
		_: Self::Message,
	) -> bool {
		true
	}

	fn view<'a>(&self, c: comp_util::Ctx<'a, Self>) -> Html {
		use super::post::ThreadPost;
		use PostSet::*;

		// TODO: Filter hidden posts

		let posts: Vec<u64> = state::read(|s| match c.props.pages {
			Last5Posts => {
				let mut v = Vec::with_capacity(5);
				let page_count = s
					.threads
					.get(&c.props.id)
					.map(|t| t.last_page + 1)
					.unwrap_or(1);
				self.read_page_posts(&mut v, c.props.id, page_count - 1, s);
				if v.len() < 5 && page_count > 1 {
					self.read_page_posts(&mut v, c.props.id, page_count - 2, s);
				}
				v.sort_unstable();
				if v.len() > 5 {
					v[v.len() - 5..].iter().copied().collect()
				} else {
					v
				}
			}
			Page(page) => {
				let mut v = Vec::with_capacity(300);
				self.read_page_posts(&mut v, c.props.id, page, s);
				v.sort_unstable();
				v
			}
		});

		html! {
			<section class="thread-container">
				<ThreadPost id=c.props.id />
				{
					for posts.into_iter().map(|id| {
						html! {
							<ThreadPost id=id />
						}
					})
				}
				// TODO: Reply button that opens a reply creation modal on both
				// the thread index and individual thread pages (allow posting
				// from thread index).
				<buttons::AsideButton
					text="reply"
					on_click=c.link.callback(|_| ())
				/>
			</section>
		}
	}
}

impl Inner {
	// Read the post IDs of a page, excluding the OP, into dst
	fn read_page_posts(
		&self,
		dst: &mut Vec<u64>,
		thread: u64,
		page: u32,
		s: &state::State,
	) {
		if let Some(posts) = s.posts_by_thread_page.get(&(thread, page)) {
			dst.extend(posts.iter().filter(|id| **id != thread));
		}
	}
}
