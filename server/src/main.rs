mod body;
mod client;
mod config;
mod db;
mod feeds;
mod message;
mod registry;
mod util;

use actix::prelude::*;
use actix_web::{get, web, App, Error, HttpRequest, HttpResponse, HttpServer};
use actix_web_actors::ws;
use registry::Registry;

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

// TODO: asset and image routes
// TODO: /robots.txt
// TODO: /health-check

#[actix_web::main]
async fn main() -> Result<(), std::io::Error> {
    // TODO: Censor DB connection string in args, if any

    async fn inner() -> util::DynResult {
        stderrlog::new().init()?;
        db::open().await?;

        // TODO: spawn global pulsar, registry and body flusher instances
        let threads = db::get_all_threads_short().await?;
        let registry = Registry::create(|ctx| Registry::new(ctx, threads));

        HttpServer::new(move || {
            App::new().app_data(registry.clone()).service(connect)
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
