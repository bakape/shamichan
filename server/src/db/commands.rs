use super::pool;
use crate::util::DynResult;

/// Increase pcount by one and return current count
pub async fn increment_pcount() -> DynResult<u64> {
	Ok(sqlx::query!(
		"insert into main as m (key, val)
		values ('pyu_count', '1')
		on conflict (key) do update
			set val = (m.val::bigint + 1)::text::jsonb
		returning val::bigint",
	)
	.fetch_one(&pool())
	.await?
	.val
	.map(|i| i as u64)
	.ok_or("pyu count not returned from query")?)
}

/// Return current pyu count
pub async fn get_pcount() -> DynResult<u64> {
	Ok(sqlx::query!(
		"select coalesce(
			(
				select val::int from main
				where key = 'pyu_count'
			),
			0
		) val",
	)
	.fetch_one(&pool())
	.await?
	.val
	.map(|i| i as u64)
	.ok_or("pyu count not returned from query")?)
}
