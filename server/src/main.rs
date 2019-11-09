mod config;

use actix_files;
use actix_web::middleware::Logger;
use actix_web::{App, HttpServer};
use env_logger;

pub fn main() {
    let server_config = config::load_server_config().unwrap();
    if cfg!(debug_assertions) {
        std::env::set_var("RUST_LOG", "actix_web=info");
    }
    env_logger::init();

    HttpServer::new(|| {
        App::new()
            .wrap(Logger::new("%a %r %s %b %T"))
            .route(
                "/",
                actix_web::web::get().to(|| {
                    actix_files::NamedFile::open("www/client/index.html")
                }),
            )
            .service(actix_files::Files::new("/assets", "www"))
    })
    .bind(server_config.listening_address.clone())
    .unwrap()
    .run()
    .unwrap();
}
