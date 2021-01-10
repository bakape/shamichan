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
use dotenv;
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
        client::Client::new(ip, registry.get_ref().clone()),
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
        db::open().await?;

        // Spawn registry on it's own thread to reduce contention
        let threads = db::get_all_threads_short().await?;
        let registry = Registry::start_in_arbiter(&Arbiter::new(), |ctx| {
            Registry::new(ctx, threads)
        });

        HttpServer::new(move || {
            use actix_files::Files;
            use actix_web::middleware::{
                normalize::TrailingSlash, Compress, Logger, NormalizePath,
            };

            App::new()
                .wrap(Logger::default())
                .wrap(NormalizePath::new(TrailingSlash::Trim))
                .wrap(Compress::default())
                .service(web::resource("/").to(|| async {
                    Index {
                        config: config::get().public.clone(),
                    }
                }))
                .data(registry.clone())
                .service(connect)
                .service(Files::new("/assets", "./www"))
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
