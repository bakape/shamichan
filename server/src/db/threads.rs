use super::{pool, PostInsertParams};
use crate::util::DynResult;
use common::payloads::{Post, ThreadWithPosts};

/// Parameters for inserting a thread and its OP
pub struct ThreadInsertParams<'a> {
	pub subject: &'a str,
	pub tags: &'a [String],
	pub op: PostInsertParams<'a>,
}

/// Insert thread and empty post into DB and return the thread ID
pub async fn insert_thread<'a>(
	p: &mut ThreadInsertParams<'a>,
) -> DynResult<u64> {
	let mut tx = pool().begin().await?;

	let id: i64 = sqlx::query!(
		"insert into threads (subject, tags)
		values ($1, $2)
		returning id",
		p.subject,
		p.tags,
	)
	.fetch_one(&mut tx)
	.await?
	.id;

	sqlx::query!(
		"insert into posts (
			id,
			thread,
			public_key,
			name,
			trip,
			flag,
			body
		)
		values (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7
		)",
		id,
		id,
		p.op.public_key.map(|i| i as i64),
		p.op.name,
		p.op.trip,
		p.op.flag,
		serde_json::to_value(p.op.body)?,
	)
	.execute(&mut tx)
	.await?;

	tx.commit().await?;

	Ok(id as u64)
}

/// Return all existing threads and their last 5 posts
pub async fn get_all_threads_short() -> DynResult<Vec<ThreadWithPosts>> {
	let mut threads: Vec<ThreadWithPosts> = Vec::new();
	let pool = pool();
	let mut s = sqlx::query!("select get_thread(id, -5) thread from threads")
		.fetch(&pool);
	while let Some(r) = s.next().await {
		if let Some(t) = r?.thread {
			threads.push(serde_json::from_value(t)?);
		}
	}

	Ok(threads)
}

/// Get a specific page of a thread
pub async fn get_page(thread: u64, page: u32) -> DynResult<Vec<Post>> {
	Ok(serde_json::from_value(
		sqlx::query!(
			"select jsonb_agg(encode(p)) page
			from posts p
			where thread = $1 and page = $2",
			thread as i64,
			page as i32,
		)
		.fetch_one(&pool())
		.await?
		.page
		.ok_or("query returned no JSON")?,
	)?)
}
