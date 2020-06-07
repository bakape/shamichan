use crate::common::DynResult;
use protocol::payloads::post_body::Node;
use std::{collections::HashMap, sync::Arc};

// Get a client corresponding to a connection from the connection pool
async fn get_client() -> DynResult<deadpool_postgres::Client> {
	// TODO: prepared statement cache

	lazy_static! {
		static ref POOL: Result<deadpool_postgres::Pool, String> =
			|| -> DynResult<deadpool_postgres::Pool> {
				use deadpool_postgres::config as cfg;

				let u: tokio_postgres::Config =
					crate::config::read(|c| c.db_url.clone()).parse()?;

				let conf = cfg::Config {
					user: u.get_user().map(|s| s.into()),
					password: u
						.get_password()
						.map(|b| {
							String::from_utf8(b.iter().copied().collect()).ok()
						})
						.flatten(),
					dbname: u.get_dbname().map(|s| s.into()),
					application_name: u
						.get_application_name()
						.map(|s| s.into()),
					ssl_mode: Some(cfg::SslMode::Disable),
					hosts: u
						.get_hosts()
						.iter()
						.map(|h| {
							use tokio_postgres::config::Host::*;

							match h {
								Tcp(s) => Ok(s.clone()),
								Unix(p) => match p.as_path().to_str() {
									Some(s) => Ok(s.into()),
									None => Err(format!(
										concat!(
											"could not parse Unix ",
											"socket host: {:?}"
										),
										h
									)
									.into()),
								},
							}
						})
						.collect::<Result<Vec<String>, String>>()
						.map(|v| if v.is_empty() { None } else { Some(v) })?,
					ports: {
						let p = u.get_ports();
						if p.is_empty() {
							None
						} else {
							Some(p.iter().copied().collect())
						}
					},
					connect_timeout: u.get_connect_timeout().cloned(),
					keepalives: u.get_keepalives().into(),
					keepalives_idle: u.get_keepalives_idle().into(),
					target_session_attrs: Some(
						cfg::TargetSessionAttrs::ReadWrite,
					),
					..Default::default()
				};

				Ok(conf.create_pool(tokio_postgres::tls::NoTls)?)
			}()
			.map_err(|e| e.to_string())
			.into();
	}

	Ok(POOL.as_ref().map_err(|e| e.clone())?.get().await?)
}

pub async fn write_open_post_bodies(
	bodies: HashMap<u64, Arc<Node>>,
) -> DynResult {
	let mut vals = bodies
		.into_iter()
		.map(|(id, body)| Ok((id, serde_json::to_vec(&body)?)))
		.collect::<Result<Vec<(u64, Vec<u8>)>, serde_json::Error>>()?;

	// Sort by ID for more sequential DB access
	vals.sort_unstable_by(|(a, _), (b, _)| a.cmp(b));

	let mut cl = get_client().await?;
	let tx = cl.transaction().await?;
	let q = tx
		.prepare(
			r#"update posts
			set body = $1
			where id = $2 and editing = true"#,
		)
		.await?;

	for (id, body) in vals {
		use tokio_postgres::types::ToSql;

		tx.execute(
			&q,
			&[
				&(id as i64) as &(dyn ToSql + Sync),
				&body as &(dyn ToSql + Sync),
			],
		)
		.await?;
	}

	tx.commit().await?;
	Ok(())
}
