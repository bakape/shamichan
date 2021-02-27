mod last_5;
mod page;
mod writer;

use super::{
	index::{Change, IndexFeed},
	FetchFeedData, InsertPost,
};
use crate::{
	body::persist_open::{BodyFlusher, PersistBodies},
	client::{Client, Disconnect, SendMessage},
	mt_context::{AsyncHandler, MTAddr, MTContext},
	registry::Registry,
	util::{self, run_in_rayon, DynResult, MessageCacher, Pulse, WakeUp},
};
use actix::prelude::*;
use async_trait::async_trait;
use common::{
	payloads::{
		post_body::Node, ImmutablePage, Post, PostCreationNotification, Thread,
	},
	MessageType,
};
use page::{MutablePage, PageRecord};
use rayon::prelude::*;
use std::{collections::HashMap, sync::Arc, time::Duration};

// TODO(?): if a feed does not have any clients and has not had activity for 5
// minutes, request the registry to deallocate this thread. This only saves
// memory and may not be worth it.

// TODO: post closing

/// Post location in a thread
#[derive(Debug, Eq, PartialEq, Hash, Clone)]
pub struct PostLocation {
	/// Page of a thread the post is on
	pub page: u32,

	/// ID of post
	pub id: u64,
}

/// Update feed. Either a thread feed or the global thread index feed.
#[derive(Debug)]
pub struct ThreadFeed {
	/// Marks this feed instance as started more than 10 seconds ago.
	/// This removed the need to query system time.
	older_than_10_seconds: bool,

	/// Last snapshot of Clients subscribed to the feed
	clients: super::Clients,

	/// Link to the global registry
	registry: Addr<Registry>,

	/// Batching open post body flusher
	body_flusher: MTAddr<BodyFlusher>,

	/// Page fetches deferred to next pulse to reduce cache thrashing
	deferred_page_fetches: HashMap<u32, Vec<Addr<Client>>>,

	/// Pending processing of buffered changes
	pending_pulse: bool,

	/// Buffering writer of messages and feed changes
	writer: writer::Writer,

	/// Open bodies pending parsing and diffing collected by page ID
	pending_open_bodies: HashMap<u32, HashMap<u64, Vec<char>>>,

	/// Thread metadata
	thread_meta: MessageCacher<Thread>,

	/// Pages currently loaded from the DB
	pages: HashMap<u32, PageRecord>,
}

impl actix::Actor for ThreadFeed {
	type Context = MTContext<Self>;

	fn started(&mut self, ctx: &mut Self::Context) {
		ctx.notify_interval(
			TryMakePagesImmutable,
			Duration::from_secs(60 * 10),
		);
	}
}

#[async_trait]
impl AsyncHandler<Pulse> for ThreadFeed {
	type Error = util::Err;

	async fn handle(
		&mut self,
		_: Pulse,
		_: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		self.pending_pulse = false;

		// Send any deferred page fetches first to maintain chronology.
		// The pages will still be sent only after the clients have received
		// thread metainformation, so they can perform negative page number
		// deduction.
		for (page, clients) in std::mem::take(&mut self.deferred_page_fetches) {
			use PageRecord::*;

			if let Some(msg) = match self.pages.get_mut(&page) {
				Some(Immutable(msg)) => Some(SendMessage(msg.clone())),
				Some(Mutable(p)) => Some(SendMessage(p.get_message()?)),
				_ => None,
			} {
				for c in clients {
					c.do_send(msg.clone());
				}
			}
		}

		self.diff_open_bodies().await?;

		self.clients = self
			.registry
			.send(crate::registry::SnapshotClients(self.thread_meta.id))
			.await?;
		self.writer.flush(self.clients.values())?;

		Ok(())
	}
}

#[async_trait]
impl AsyncHandler<WakeUp> for ThreadFeed {
	type Error = ();

	async fn handle(
		&mut self,
		_: WakeUp,
		ctx: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		self.schedule_pulse(ctx);
		Ok(())
	}
}

/// Request to fetch a page from the thread and send it to the client
pub struct FetchPage {
	/// Page ID to fetch.
	/// Negative numbers count from the end.
	pub id: i32,

	/// Client to send the page to
	pub client: Addr<Client>,
}

#[async_trait]
impl AsyncHandler<FetchPage> for ThreadFeed {
	type Error = util::Err;

	async fn handle(
		&mut self,
		FetchPage { mut id, client }: FetchPage,
		ctx: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		use PageRecord::*;

		if id < 0 {
			id = self.thread_meta.page_count as i32 + id;
		}
		{
			let max = self.thread_meta.page_count as i32 - 1;
			if id < 0 || id > max {
				client.do_send(Disconnect(
					format!("requested page out of bounds: {}", id).into(),
				));
				return Ok(());
			}
		}
		if id as u64 > (std::u32::MAX >> 1) as u64 {
			return Err("page ID overflow".into());
		}
		let id = id as u32;

		// TODO: increment spam score. This is a mildly expensive operation.

		match self
			.pages
			.get_mut(&id)
			.ok_or_else(|| format!("page not inserted: {}", id))
			.unwrap()
		{
			p @ Unfetched => {
				let mut page =
					Self::fetch_page(self.thread_meta.id, id).await?;
				client.do_send(SendMessage(match &mut page {
					Unfetched => unreachable!(),
					Mutable(p) => p.get_message()?,
					Immutable(msg) => msg.clone(),
				}));
				*p = page;
			}
			Mutable(p) => {
				// Reply immediately, if cached. If not, defer to reduce
				// cache thrashing.
				match p.get_cached_message() {
					Some(m) => {
						client.do_send(SendMessage(m.clone()));
					}
					None => {
						self.schedule_pulse(ctx);
						self.deferred_page_fetches
							.entry(id)
							.or_default()
							.push(client);
					}
				};
			}
			Immutable(p) => {
				client.do_send(SendMessage(p.clone()));
			}
		};
		Ok(())
	}
}

#[async_trait]
impl AsyncHandler<InsertPost> for ThreadFeed {
	type Error = util::Err;

	async fn handle(
		&mut self,
		req: InsertPost,
		ctx: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		use page::PageRecord::*;
		use std::collections::hash_map::Entry;

		self.schedule_pulse(ctx);
		if req.page > self.thread_meta.page_count {
			self.thread_meta.page_count = req.page;
		}
		let now = util::now();
		if !req.opts.sage {
			self.thread_meta.bumped_on = now;
		}

		let payload = PostCreationNotification {
			id: req.id,
			page: req.page,
			thread: self.thread_meta.id,
			time: now,
		};
		let post =
			Post::new(req.id, req.thread, req.page, now, req.opts.clone());

		#[rustfmt::skip]
		macro_rules! insert_post {
			($page:expr) => {
				$page.insert(req.id, post.clone().into());
				self.writer.register_post_id(req.id);
				self.writer.write_message(MessageType::InsertPost, &payload)?;
				self.writer.write_global_change(
					MessageType::InsertPost,
					&payload,
					Change::InsertPost(post),
				)?;
			};
		}

		macro_rules! err_immutable {
			() => {
				return Err(
					format!(
						"trying to insert post {} into immutable page {} in thread {}",
						req.id,
						req.page,
						req.thread
					)
					.into(),
				);
			};
		}

		match self.pages.entry(req.page) {
			Entry::Occupied(mut e) => match e.get_mut() {
				Immutable(_) => {
					err_immutable!();
				}
				Mutable(p) => {
					insert_post!(p);
				}
				p @ Unfetched => {
					let mut page =
						Self::fetch_page(self.thread_meta.id, req.page).await?;
					match &mut page {
						Unfetched => {
							unreachable!();
						}
						Immutable(_) => {
							err_immutable!();
						}
						Mutable(p) => {
							insert_post!(p);
						}
					};
					*p = page;
				}
			},
			Entry::Vacant(e) => {
				e.insert(PageRecord::new_mutable(Some(post)));
			}
		}
		Ok(())
	}
}

/// Set the text body of an open post
pub struct SetBody {
	pub loc: PostLocation,
	pub body: Vec<char>,
}

#[async_trait]
impl AsyncHandler<SetBody> for ThreadFeed {
	type Error = ();

	async fn handle(
		&mut self,
		SetBody { loc, body }: SetBody,
		ctx: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		self.schedule_pulse(ctx);
		self.pending_open_bodies
			.entry(loc.page)
			.or_default()
			.insert(loc.id, body);
		Ok(())
	}
}

/// Try to make any pages that can no longer change immutable by moving them
/// to memory-mapped files
#[derive(Clone)]
struct TryMakePagesImmutable;

#[async_trait]
impl AsyncHandler<TryMakePagesImmutable> for ThreadFeed {
	type Error = util::Err;

	async fn handle(
		&mut self,
		_: TryMakePagesImmutable,
		_: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		for (id, rec) in self.pages.iter_mut() {
			if let PageRecord::Mutable(p) = rec {
				if PageRecord::can_be_made_immutable(
					p.values().map(|p| p.as_ref()),
				) {
					let mut old = PageRecord::Unfetched;
					std::mem::swap(&mut old, rec);
					match old {
						PageRecord::Mutable(mut p) => {
							*rec = PageRecord::new_immutable(&ImmutablePage {
								thread: self.thread_meta.id,
								page: *id,
								posts: p
									.drain()
									.map(|(_, p)| p.get())
									.collect(),
							})
							.await?;
						}
						_ => unreachable!(),
					}
				}
			}
		}
		Ok(())
	}
}

#[async_trait]
impl AsyncHandler<FetchFeedData> for ThreadFeed {
	type Error = util::Err;

	async fn handle(
		&mut self,
		FetchFeedData(client): FetchFeedData,
		_: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		client.do_send(SendMessage(
			self.thread_meta.get_message(MessageType::ThreadMeta)?,
		));
		Ok(())
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
		thread: Thread,
		last_5_posts: impl IntoIterator<Item = u64>,
		first_page: Option<Vec<Post>>,
		registry: Addr<Registry>,
		index_feed: MTAddr<IndexFeed>,
		body_flusher: MTAddr<BodyFlusher>,
	) -> Self {
		let mut f = Self {
			registry,
			body_flusher,
			pending_pulse: false,
			older_than_10_seconds: false,
			clients: Default::default(),
			writer: writer::Writer::new(thread.id, index_feed, last_5_posts),
			thread_meta: thread.into(),
			pending_open_bodies: Default::default(),
			deferred_page_fetches: Default::default(),
			pages: Default::default(),
		};

		for i in 0..=f.thread_meta.page_count {
			f.pages.insert(i, PageRecord::Unfetched);
		}
		if let Some(p) = first_page {
			f.pages.insert(0, PageRecord::new_mutable(p));
		}

		f
	}

	/// Schedule processing of buffered changes
	fn schedule_pulse(&mut self, ctx: &mut <Self as Actor>::Context) {
		if !self.pending_pulse {
			self.pending_pulse = true;
			ctx.notify_later(Pulse, Duration::from_millis(100));
		}
	}

	/// Request to fetch an existing page from the database.
	/// Static function to avoid referencing self.
	async fn fetch_page(thread: u64, page: u32) -> DynResult<PageRecord> {
		let posts = crate::db::get_page(thread, page).await?;
		Ok(if PageRecord::can_be_made_immutable(posts.iter()) {
			PageRecord::new_immutable(&ImmutablePage {
				thread,
				page,
				posts,
			})
			.await?
		} else {
			PageRecord::new_mutable(posts)
		})
	}

	/// Diff pending open post body changes in parallel and write messages to
	/// encoders
	async fn diff_open_bodies(&mut self) -> DynResult {
		use common::payloads::post_body::{PatchNode, PostBodyPatch};

		if self.pending_open_bodies.len() == 0 {
			return Ok(());
		}

		async fn process(
			page: &mut MutablePage,
			pending: HashMap<u64, Vec<char>>,
			mutation_batch: &mut Vec<(u64, PatchNode, Arc<Node>)>,
		) -> DynResult {
			let to_diff = pending
				.into_iter()
				.filter_map(|(id, body)| {
					page.get(&id)
						.map(|p| {
							if p.open {
								Some((id, p.body.clone(), body))
							} else {
								None
							}
						})
						.flatten()
				})
				.collect::<Vec<_>>();
			for (id, body, patch) in run_in_rayon(move || {
				to_diff
					.into_par_iter()
					.filter_map(|(id, old, new)| {
						match crate::body::parse(
							&new.into_iter().collect::<String>(),
							true,
						) {
							Ok(new) => old
								.diff(&new)
								.map(|patch| (id, Arc::new(new), patch)),
							Err(e) => {
								log::error!(
									"failed to parse post {} body: {}",
									id,
									e
								);
								None
							}
						}
					})
					.collect::<Vec<_>>()
			})
			.await?
			.into_iter()
			{
				page.get_mut(&id).unwrap().body = body.clone();
				mutation_batch.push((id, patch, body));
			}

			Ok(())
		}

		let mut mutation_batch = Vec::<(u64, PatchNode, Arc<Node>)>::new();
		for (page_id, pending) in std::mem::take(&mut self.pending_open_bodies)
		{
			use PageRecord::*;

			match self.pages.get_mut(&page_id) {
				Some(p @ Unfetched) => {
					let mut page =
						Self::fetch_page(self.thread_meta.id, page_id).await?;
					if let Mutable(p) = &mut page {
						process(p, pending, &mut mutation_batch).await?;
					}
					*p = page;
				}
				Some(Mutable(p)) => {
					process(p, pending, &mut mutation_batch).await?;
				}
				_ => (),
			};
		}
		if !mutation_batch.is_empty() {
			let req = PersistBodies(
				mutation_batch
					.into_iter()
					.map(|(id, patch, body)| {
						self.writer.write_post_message(
							id,
							MessageType::PatchPostBody,
							&PostBodyPatch { id, patch },
							Change::SetBody {
								id,
								body: body.clone(),
							},
						)?;
						Ok((id, body))
					})
					.collect::<DynResult<Vec<_>>>()?,
			);
			self.body_flusher.do_send(req);
		}

		Ok(())
	}
}
