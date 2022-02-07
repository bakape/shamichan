mod body;
mod client;
mod config;
mod db;
mod feeds;
mod message;
mod mt_context;
mod registry;
mod util;

use actix::prelude::*;
use actix_web::{get, web, App, Error, HttpRequest, HttpResponse, HttpServer};
use actix_web_actors::ws;
use askama::Template;
use cfg_if::cfg_if;
use dotenv;
use feeds::IndexFeed;
use mt_context::MTAddr;
use registry::Registry;
use std::sync::Arc;

// TODO: asset routes
// TODO: /robots.txt
// TODO: /health-check

// TODO: ETag support via hashing the source HTML and configuration
#[derive(Template)]
#[template(path = "index.html")]
struct Index {
	config: Arc<common::config::Public>,
}

#[get("/api/socket")]
async fn connect(
	req: HttpRequest,
	stream: web::Payload,
	registry: web::Data<Addr<Registry>>,
	index_feed: web::Data<MTAddr<IndexFeed>>,
) -> Result<HttpResponse, Error> {
	let ci = req.connection_info();
	let ip = if config::SERVER.reverse_proxied {
		ci.realip_remote_addr()
	} else {
		ci.peer_addr()
	}
	.map(|s| s.parse::<std::net::SocketAddr>().ok())
	.flatten()
	.map(|a| a.ip())
	.ok_or(actix_web::error::ErrorBadRequest(
		"could not read client IP",
	))?;

	ws::start(
		client::Client::new(
			ip,
			registry.get_ref().clone(),
			index_feed.get_ref().clone(),
		),
		&req,
		stream,
	)
}

#[actix_web::main]
async fn main() -> Result<(), std::io::Error> {
	async {
		match dotenv::dotenv() {
			Ok(_) => (),
			Err(e) => {
				// Ignore missing .env file
				if !e.not_found() {
					Err(e)?
				}
			}
		};

		stderrlog::new()
			.timestamp(stderrlog::Timestamp::Millisecond)
			.verbosity(match config::SERVER.log_level {
				log::Level::Error => 0,
				log::Level::Warn => 1,
				log::Level::Info => 2,
				log::Level::Debug => 3,
				log::Level::Trace => 4,
			})
			.init()?;

		// Set's a more descriptive process title - only port number because of
		// 12 char limit.
		// Also prevents exposing DB connection string in args, if any.
		proctitle::set_title(format!(
			"shamichan @ :{}",
			config::SERVER
				.address
				.parse::<std::net::SocketAddr>()?
				.port(),
		));

		// TODO: remove this and revert tokio runtime to private, when we switch
		// to actix_web, askama and actix_web_actors to 4.0.
		let threads = mt_context::TOKIO_RUNTIME.block_on(async {
			db::open().await?;

			db::get_all_threads_short().await
		})?;

		// Might as well register them to remove the need to fetch them later
		body::cache_locations(
			threads.iter().map(|t| t.posts.values()).flatten(),
		);

		// Spawn registry on it's own thread to reduce contention
		let registry =
			Registry::start_in_arbiter(&Arbiter::new().handle(), move |ctx| {
				Registry::new(ctx, threads)
			});
		let index_feed = registry.send(registry::GetIndexFeed).await?;

		let s = HttpServer::new(move || {
			use actix_files::Files;
			use actix_web::middleware::{
				Compress, Logger, NormalizePath, TrailingSlash,
			};

			cfg_if! {
				if #[cfg(debug_assertions)] {
					let app = App::new().wrap_fn(|req, srv| {
						use actix_service::Service;
						use actix_web::http::{
							header::{CACHE_CONTROL, HeaderValue},
						};

						let fut = srv.call(req);
						async {
							let mut res = fut.await?;
							res.headers_mut().insert(
								CACHE_CONTROL,
								HeaderValue::from_static("no-store"),
							);
							Ok(res)
						}
					});
				} else {
					let app = App::new();
				}
			};
			let mut app = app
				.wrap(Logger::default())
				.wrap(NormalizePath::new(TrailingSlash::Trim))
				.wrap(Compress::default())
				.app_data(registry.clone())
				.app_data(index_feed.clone())
				.service(connect)
				.service(Files::new("/assets", "./www"));

			for p in &["/", "/catalog", "/threads/{thread:\\d+}/{page:\\d+}"] {
				app = app.service(web::resource(*p).to(|| async {
					let i = Index {
						config: config::get().public.clone(),
					};
					match i.render() {
						Ok(s) => s,
						Err(e) => e.to_string(),
					}
				}));
			}

			app
		})
		.bind(&config::SERVER.address)?;

		log::info!("server started on http://{}", config::SERVER.address);

		s.run().await?;

		Ok::<(), util::Err>(())
	}
	.await
	.map_err(|err| {
		std::io::Error::new(std::io::ErrorKind::Other, err.to_string())
	})
}
