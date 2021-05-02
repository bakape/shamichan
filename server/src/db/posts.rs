use super::pool;
use crate::util::DynResult;
use common::payloads::post_body::Node;
use std::{collections::HashMap, sync::Arc};

// Common params for both post and thread insertion
pub struct PostInsertParams<'a> {
	pub public_key: Option<u64>,
	pub name: Option<&'a str>,
	pub trip: Option<&'a str>,
	pub flag: Option<&'a str>,
	pub body: &'a Node,
}

pub async fn write_open_post_bodies(
	bodies: HashMap<u64, Arc<Node>>,
) -> DynResult {
	let mut bodies: Vec<(u64, Arc<Node>)> = bodies.into_iter().collect();

	// Sort by ID for more sequential DB access
	bodies.sort_unstable_by(|(a, _), (b, _)| a.cmp(b));

	let mut tx = pool().begin().await?;
	for (id, body) in bodies {
		sqlx::query!(
			"update posts
			set body = $1
			where id = $2 and open = true",
			serde_json::to_value(body)?,
			id as i64,
		)
		.execute(&mut tx)
		.await?;
	}
	tx.commit().await?;
	Ok(())
}

/// Close open post and set its body
pub async fn close_post(id: u64, body: &Node) -> DynResult {
	sqlx::query!(
		"update posts
		set
			open = false,
			body = $2
		where id = $1",
		id as i64,
		serde_json::to_value(body)?,
	)
	.execute(&pool())
	.await?;

	Ok(())
}

/// Insert post into the database and return its ID and page
pub async fn insert_post<'a>(
	thread: u64,
	sage: bool,
	p: &PostInsertParams<'a>,
) -> DynResult<(u64, u32)> {
	let r = sqlx::query!(
		"insert into posts (
			thread,
			public_key,
			name,
			trip,
			flag,
			sage,
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
		)
		returning id, page",
		thread as i64,
		p.public_key.map(|i| i as i64),
		p.name,
		p.trip,
		p.flag,
		sage,
		serde_json::to_value(p.body)?,
	)
	.fetch_one(&pool())
	.await?;

	Ok((r.id as u64, r.page as u32))
}

/// Return the thread and page of a post, if any
pub async fn get_post_parenthood(
	id: u64,
) -> Result<Option<(u64, u32)>, sqlx::Error> {
	Ok(sqlx::query!(
		"select thread, page
		from posts
		where id = $1",
		id as i64,
	)
	.fetch_optional(&pool())
	.await?
	.map(|r| (r.thread as u64, r.page as u32)))
}
