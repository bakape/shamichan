use super::{
	index::{Change, IndexFeed},
	Feed, FeedCommon,
};
use crate::{
	body::persist_open::{BodyFlusher, PersistBodies},
	message::Message,
	registry::Registry,
	util::{self, DynResult, MessageCacher},
};
use actix::prelude::*;
use bytes::Bytes;
use common::{
	payloads::{post_body::Node, Page, Post, Thread},
	Encoder, MessageType,
};
use futures::future::FutureExt;
use rayon::prelude::*;
use serde::Serialize;
use std::{collections::HashMap, sync::Arc};

// TODO: schedule pulse on all request

/// Holds the IDs of up to the last 5 posts
type Last5Posts =
	heapless::BinaryHeap<u64, heapless::consts::U5, heapless::binary_heap::Min>;

/// Contains thread page data
#[derive(Debug)]
enum PageRecord {
	/// Contains open posts that can still be edited or is bellow the
	/// page capacity of 100
	Mutable(MessageCacher<Page>),

	/// Does not contain any open posts and is at full page capacity
	Immutable(Bytes),
}

impl PageRecord {
	async fn new(page: Page) -> DynResult<Self> {
		Ok(if !Self::can_be_made_immutable(&page) {
			Self::Mutable(page.into())
		} else {
			Self::new_immutable(&Encoder::encode(MessageType::Page, &page)?)
				.await?
		})
	}

	/// Construct new immutable PageRecord
	async fn new_immutable(buf: impl AsRef<[u8]>) -> DynResult<Self> {
		let buf = buf.as_ref();
		let len = buf.len();
		let mut m = Box::new(
			actix_web::web::block(move || {
				memmap::MmapOptions::new().len(len).map_anon()
			})
			.await?,
		);
		m.as_mut().clone_from_slice(&buf);

		// XXX: This leaks virtual memory, if the thread is deleted.
		// To not leak the mmap you'd need to extend Bytes with construction
		// from an owned AsRef<[u8]> (AsRef<[u8]> + Drop ?) or just MMap and
		// MMapMut, and have it drop the memory map, when Bytes is dropped.
		Ok(Self::Immutable(Bytes::from_static(Box::leak(m))))
	}

	/// Returns if a page can be considered immutable
	fn can_be_made_immutable(page: &Page) -> bool {
		page.posts.len() == 100 && page.posts.values().all(|p| !p.open)
	}
}

/// Pending messages and changes to be sent to the global thread index feed
#[derive(Debug, Default)]
struct GlobalPending {
	/// Messages to be encoded and sent
	enc: Encoder,

	/// Changes reflected by the messages
	changes: Vec<Change>,
}

/// Post location in a thread
#[derive(Debug, Eq, PartialEq, Hash, Clone)]
pub struct PostLocation {
	pub page: u32,
	pub id: u64,
}

/// Update feed. Either a thread feed or the global thread index feed.
#[derive(Debug)]
pub struct ThreadFeed {
	common: FeedCommon<Self>,

	/// Link to the thread index ThreadFeed
	index_feed: Addr<IndexFeed>,

	/// Batching open post body flusher
	body_flusher: Addr<BodyFlusher>,

	/// Pending message streaming encoder
	pending: Option<Encoder>,

	/// Pending messages and changes to be sent to the global thread index feed
	global_pending: Option<GlobalPending>,

	/// Last 5 post IDs in the thread
	last_5_posts: Last5Posts,

	/// Open bodies pending parsing and diffing by (page_id, post_id)
	pending_open_bodies: HashMap<PostLocation, Vec<char>>,

	/// Thread metadata
	thread_meta: MessageCacher<Thread>,

	/// Periodically tries to make pages immutable
	page_cleanup: SpawnHandle,

	/// Pages currently loaded from the DB
	pages: HashMap<u32, PageRecord>,
}

crate::impl_feed_commons! {ThreadFeed}

impl Feed for ThreadFeed {
	fn get_feed_common(&mut self) -> &mut FeedCommon<Self> {
		&mut self.common
	}

	fn process_pulse(&mut self) {
		todo!("process all pending state, notify clients");
		todo!("notify the index feed using do_send()")
	}
}

impl Handler<super::InsertPost> for ThreadFeed {
	type Result = ();

	fn handle(
		&mut self,
		req: super::InsertPost,
		ctx: &mut Self::Context,
	) -> Self::Result {
		self.schedule_pulse(ctx);
		self.push_to_last_5(req.id);

		let payload = common::payloads::PostCreationNotification {
			id: req.id,
			page: req.page,
			thread: self.thread_meta.id,
			time: util::now(),
		};
		self.write_message(MessageType::InsertPost, &payload);
		self.write_global_change(
			MessageType::InsertPost,
			&payload,
			Change::InsertPost(req),
		);
	}
}

/// Set the text body of an open post
#[derive(Message)]
#[rtype(result = "()")]
pub struct SetBody {
	pub loc: PostLocation,
	pub body: Vec<char>,
}

impl Handler<SetBody> for ThreadFeed {
	type Result = ();

	fn handle(&mut self, req: SetBody, _: &mut Self::Context) -> Self::Result {
		self.pending_open_bodies.insert(req.loc, req.body);
	}
}

impl ThreadFeed {
	/// Create a new ThreadFeed.
	///
	/// `last_5_posts` can contain more than just the IDs of the last 5 posts.
	///
	/// `first_page` can optionally be provided, if known, to avoid a page fetch
	/// from the DB.
	pub fn new(
		ctx: &mut <Self as Actor>::Context,
		thread: Thread,
		last_5_posts: impl IntoIterator<Item = u64>,
		first_page: Option<Page>,
		registry: Addr<Registry>,
		index_feed: Addr<IndexFeed>,
		body_flusher: Addr<BodyFlusher>,
	) -> Self {
		let mut f = Self {
			common: FeedCommon::new(registry),
			thread_meta: thread.into(),
			index_feed,
			body_flusher,
			last_5_posts: Default::default(),
			pending: Default::default(),
			global_pending: Default::default(),
			pending_open_bodies: Default::default(),
			pages: Default::default(),
			page_cleanup: ctx.run_interval(
				std::time::Duration::from_secs(60 * 10),
				|this, ctx| this.try_make_pages_immutable(ctx),
			),
		};

		for id in last_5_posts {
			f.push_to_last_5(id);
		}

		if let Some(p) = first_page {
			f.pages.insert(0, PageRecord::Mutable(p.into()));
		}

		f
	}

	/// Push post ID to last 5 post heap
	fn push_to_last_5(&mut self, id: u64) {
		if self.last_5_posts.len() == 5 {
			unsafe { self.last_5_posts.pop_unchecked() };
		}
		unsafe { self.last_5_posts.push_unchecked(id) };
	}

	/// This should never happen, but log it and halt execution, if it does.
	/// Caller should abort execution.
	fn log_encode_error(&self, err: std::io::Error) {
		log::error!(
			"could not encode message on feed {}: {:?}",
			self.thread_meta.id,
			err
		);
	}

	/// Schedule buffered state processing
	fn schedule_pulse(&mut self, ctx: &mut <Self as Actor>::Context) {
		self.common.schedule_pulse(ctx);
	}

	/// Write post-related message to pending message encoder and propagate it
	/// to the global feed together with `change`, if needed.
	fn write_post_message(
		&mut self,
		post_id: u64,
		t: MessageType,
		payload: &impl Serialize,
		change: Change,
	) {
		self.write_message(t, payload);
		if self.include_in_global(post_id) {
			self.write_global_change(t, payload, change);
		}
	}

	/// Write message to pending message encoder
	fn write_message(&mut self, t: MessageType, payload: &impl Serialize) {
		if let Err(e) = match &mut self.pending {
			Some(e) => e,
			None => {
				self.pending = Some(Default::default());
				self.pending.as_mut().unwrap()
			}
		}
		.write_message(t, payload)
		{
			self.log_encode_error(e);
		}
	}

	/// Write message and change to the pending global feed changeset
	fn write_global_change(
		&mut self,
		t: MessageType,
		payload: &impl Serialize,
		change: Change,
	) {
		let set = match &mut self.global_pending {
			Some(s) => s,
			None => {
				self.global_pending = Some(Default::default());
				self.global_pending.as_mut().unwrap()
			}
		};
		if let Err(e) = set.enc.write_message(t, payload) {
			self.log_encode_error(e);
			return;
		}
		set.changes.push(change);
	}

	/// Return, if post should be included in the global thread index
	fn include_in_global(&self, id: u64) -> bool {
		id == self.thread_meta.id
			|| self
				.last_5_posts
				.peek()
				.map(|lp| &id >= lp)
				.unwrap_or(false)
	}

	/// Get a post still able to be mutated by ID.
	/// This function fails for posts from immutable pages.
	fn get_mutable_post(
		&mut self,
		loc: &PostLocation,
	) -> Result<&mut Post, String> {
		match self.pages.get_mut(&loc.page) {
			Some(PageRecord::Mutable(page)) => {
				match page.posts.get_mut(&loc.id) {
					Some(p) => Ok(p),
					None => Err(format!(
						"post {} not found on page {}",
						loc.id, loc.page
					)),
				}
			}
			Some(PageRecord::Immutable(_)) => Err(format!(
				"trying to retrieve post {} from immutable page {}",
				loc.id, loc.page
			)),
			None => Err(format!(
				"trying to retrieve post {} from missing page {}",
				loc.id, loc.page
			)),
		}
	}

	/// Diff pending open post body changes in parallel and write messages to
	/// encoders
	async fn diff_open_bodies(&mut self) -> DynResult {
		if self.pending_open_bodies.is_empty() {
			return Ok(());
		}

		let to_diff: Vec<(PostLocation, Arc<Node>, String)> =
			std::mem::take(&mut self.pending_open_bodies)
				.into_iter()
				.filter_map(|(loc, body)| match self.get_mutable_post(&loc) {
					Ok(p) => {
						if p.open {
							Some(Ok((
								loc.clone(),
								p.body.clone(),
								body.into_iter().collect(),
							)))
						} else {
							None
						}
					}
					Err(e) => Some(Err(e)),
				})
				.collect::<Result<Vec<_>, String>>()?;

		let mut mutation_batch =
			Vec::<(u64, Arc<Node>)>::with_capacity(to_diff.len());
		for (loc, res) in actix_web::web::block::<_, _, ()>(|| {
			Ok(to_diff
				.into_par_iter()
				.map(|(loc, old, body)| {
					use crate::body::{diff, parse};

					match parse(&body, true) {
						Ok(new) => (
							loc,
							Ok(diff(&old, &new)
								.map(|patch| (Arc::new(new), patch))),
						),
						Err(e) => (loc, Err(e)),
					}
				})
				.collect::<Vec<_>>())
		})
		.await?
		{
			match res {
				Ok(Some((body, patch))) => {
					self.get_mutable_post(&loc)?.body = body.clone();
					self.write_post_message(
						loc.id,
						MessageType::PatchPostBody,
						&common::payloads::post_body::PostBodyPatch {
							id: loc.id,
							patch,
						},
						Change::SetBody {
							id: loc.id,
							body: body.clone(),
						},
					);
					mutation_batch.push((loc.id, body));
				}
				Ok(None) => (),
				Err(e) => {
					log::error!("error diffing post {}: {}", loc.id, e);
				}
			}
		}

		self.body_flusher.send(PersistBodies(mutation_batch)).await;

		Ok(())
	}

	/// Try to make any pages that can no longer change immutable by moving them
	/// to memory-mapped files
	fn try_make_pages_immutable(&mut self, ctx: &mut <Self as Actor>::Context) {
		let to_make_immutable: Vec<(u32, Message)> = match self
			.pages
			.iter_mut()
			.filter_map(|(id, p)| match p {
				PageRecord::Immutable(_) => None,
				PageRecord::Mutable(p) => {
					if PageRecord::can_be_made_immutable(p) {
						Some(
							p.get_message(MessageType::Page)
								.map(|msg| (*id, msg)),
						)
					} else {
						None
					}
				}
			})
			.collect()
		{
			Ok(v) => v,
			Err(e) => {
				self.log_encode_error(e);
				return;
			}
		};

		futures::future::join_all(to_make_immutable.into_iter().map(
			|(id, msg)| {
				PageRecord::new_immutable(msg).map(move |res| (id, res))
			},
		))
		.into_actor(self)
		.then(|res, this, _| {
			for (id, res) in res {
				match res {
					Ok(page) => {
						this.pages.insert(id, page);
					}
					Err(err) => {
						log::error!(
							"failed to make page {} of thread {} immutable: {}",
							id,
							this.thread_meta.id,
							err
						);
					}
				}
			}

			fut::ready(())
		})
		.wait(ctx);
	}
}
