use super::pool;
use crate::util::DynResult;
use rand::prelude::*;
use uuid::Uuid;

/// Write public key to DB, if not already written.
/// Return its private and public IDs and, if this was a fresh insert or an
/// existing key.
pub async fn register_public_key(
	pub_key: &[u8],
) -> DynResult<(u64, Uuid, bool)> {
	let pub_id: Uuid = {
		let mut buf: [u8; 16] = Default::default();
		thread_rng().try_fill_bytes(&mut buf)?;
		uuid::Builder::from_slice(&buf)?.build()
	};

	// Perform upsert attempt first to ensure public key is always in the DB by
	// the time the select is executed

	let fresh = sqlx::query!(
		"insert into public_keys (public_id, public_key)
		values ($1, $2)
		on conflict (public_key) do nothing",
		pub_id,
		pub_key,
	)
	.execute(&pool())
	.await?
	.rows_affected()
		== 1;

	let r = sqlx::query!(
		"select id, public_id
		from public_keys
		where public_key = $1",
		pub_key,
	)
	.fetch_one(&pool())
	.await?;

	Ok((r.id as u64, r.public_id, fresh))
}

/// Get public key's private ID and key buffer by its public ID
pub async fn get_public_key(
	pub_id: &Uuid,
) -> Result<Option<(u64, Vec<u8>)>, sqlx::Error> {
	sqlx::query!(
		"select id, public_key
		from public_keys
		where public_id = $1",
		pub_id,
	)
	.fetch_optional(&pool())
	.await
	.map(|r| r.map(|r| (r.id as u64, r.public_key)))
}
