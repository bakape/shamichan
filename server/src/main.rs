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
use askama_actix::Template;
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
        ci.remote_addr()
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
    dotenv::dotenv().ok();

    // TODO: Censor DB connection string in args, if any

    async fn inner() -> util::DynResult {
        stderrlog::new().init()?;

        // TODO: remove this and revert tokio runtime to private, when we switch
        // to actix_web, askama and actix_web_actors to 4.0.
        let threads = mt_context::TOKIO_RUNTIME.block_on(async {
            db::open().await?;

            db::get_all_threads_short().await
        })?;

        // Spawn registry on it's own thread to reduce contention
        let registry =
            Registry::start_in_arbiter(&Arbiter::new(), move |ctx| {
                Registry::new(ctx, threads)
            });
        let index_feed = registry.send(registry::GetIndexFeed).await?;

        HttpServer::new(move || {
            use actix_files::Files;
            use actix_web::middleware::{
                normalize::TrailingSlash, Compress, Logger, NormalizePath,
            };

            cfg_if! {
                if #[cfg(debug_assertions)] {
                    let app = App::new().wrap_fn(|req, srv| {
                        use actix_service::Service;
                        use actix_web::http::{
                            header::CACHE_CONTROL,
                            HeaderValue,
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
                .data(registry.clone())
                .data(index_feed.clone())
                .service(connect)
                .service(Files::new("/assets", "./www"));

            for p in &["/", "/catalog", "/threads/{thread:\\d+}/{page:\\d+}"] {
                app = app.service(web::resource(*p).to(|| async {
                    Index {
                        config: config::get().public.clone(),
                    }
                }));
            }

            app
        })
        .bind(&config::SERVER.address)?
        .run()
        .await?;
        Ok(())
    }

    inner().await.map_err(|err| {
        std::io::Error::new(std::io::ErrorKind::Other, err.to_string())
    })
}
