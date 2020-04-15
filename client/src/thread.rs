use super::buttons;
use super::state;
use yew::{html, Component, ComponentLink, Html, Properties};

// Central thread container
pub struct Thread {
	#[allow(unused)]
	bridge: state::HookBridge,

	#[allow(unused)]
	link: ComponentLink<Self>,

	id: u64,
	pages: PostSet,
}

// Posts to display in a thread
#[derive(Clone)]
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

#[derive(Clone, Properties)]
pub struct Props {
	pub id: u64,
	pub pages: PostSet,
}

impl Component for Thread {
	type Message = ();
	type Properties = Props;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			bridge: state::hook(
				&link,
				&[state::Change::Thread(props.id)],
				|_| (),
			),
			id: props.id,
			pages: props.pages,
			link,
		}
	}

	fn update(&mut self, _: Self::Message) -> bool {
		true
	}

	fn view(&self) -> Html {
		// TODO: Filter hidden posts

		let posts: Vec<u64> = state::read(|s| match self.pages {
			PostSet::Last5Posts => {
				let mut v = Vec::with_capacity(5);
				let page_count = s
					.threads
					.get(&self.id)
					.map(|t| t.last_page + 1)
					.unwrap_or(1);
				self.read_page_posts(&mut v, page_count - 1, s);
				if v.len() < 5 && page_count > 1 {
					self.read_page_posts(&mut v, page_count - 2, s);
				}
				v.sort_unstable();
				if v.len() > 5 {
					v[v.len() - 5..].iter().copied().collect()
				} else {
					v
				}
			}
			PostSet::Page(page) => {
				let mut v = Vec::with_capacity(300);
				self.read_page_posts(&mut v, page, s);
				v.sort_unstable();
				v
			}
		});

		html! {
			<section class="thread-container">
				<super::post::Post id=self.id />
				{
					for posts.into_iter().map(|id| {
						html! {
							<super::post::Post id={id} />
						}
					})
				}
				// TODO: Reply button that opens a reply creation modal on both
				// the thread index and individual thread pages (allow posting
				// from thread index).
				<buttons::AsideButton
					text="reply"
					on_click=self.link.callback(|_| ())
				/>
			</section>
		}
	}
}

impl Thread {
	// Read the post IDs of a page, excluding the OP, into dst
	fn read_page_posts(&self, dst: &mut Vec<u64>, page: u32, s: &state::State) {
		if let Some(posts) = s.posts_by_thread_page.get(&(self.id, page)) {
			dst.extend(posts.iter().filter(|id| **id != self.id));
		}
	}
}
