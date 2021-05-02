mod auth;
mod commands;
mod posts;
mod threads;

pub use auth::*;
pub use commands::*;
pub use posts::*;
pub use threads::*;

use crate::util::DynResult;
use sqlx::postgres::PgPool;

static mut POOL: Option<PgPool> = None;

/// Open database connection pool
#[cold]
pub async fn open() -> DynResult {
	let pool = sqlx::postgres::PgPoolOptions::new()
		.max_connections(128)
		.connect(&crate::config::SERVER.database)
		.await?;
	sqlx::migrate!("../migrations").run(&pool).await?;
	unsafe { POOL = Some(pool) };
	Ok(())
}

/// Get a handle on the connection pool
#[inline]
fn pool() -> PgPool {
	unsafe { POOL.clone().unwrap() }
}
