mod auth;
mod posts;
mod threads;

pub use auth::*;
pub use posts::*;
pub use threads::*;

use crate::util::DynResult;
use sqlx::postgres::PgPool;

static mut POOL: Option<PgPool> = None;

/// Open database connection pool
pub async fn open() -> DynResult {
	let url = &crate::config::SERVER.database;

	// TODO: PR MigrationSource impl for Dir and run migrations at runtime as
	// well.
	// use include_dir::{include_dir, Dir};
	// static MIGRATIONS: Dir = include_dir!("../migrations");
	// sqlx::migrate::Migrator::new(&MIGRATIONS).run(&url).await?;

	unsafe {
		POOL = Some(
			sqlx::postgres::PgPoolOptions::new()
				.max_connections(128)
				.connect(url)
				.await?,
		)
	};

	Ok(())
}

/// Get a handle on the connection pool
fn pool() -> PgPool {
	unsafe { POOL.clone().unwrap() }
}
