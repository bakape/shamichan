# TODO: separate server image from imager image
# TODO: switch to focal for both and remove ffmpeg and webp building
# TODO: coverage report in the CI
FROM debian:buster

EXPOSE 8000

RUN mkdir -p /meguca/images /meguca/www/videos /src
CMD ["-a", ":8000"]
ENTRYPOINT ["./meguca"]

# Install OS dependencies
ENV DEBIAN_FRONTEND=noninteractive
RUN echo deb-src \
	http://ftp.debian.org/debian/ \
	buster main contrib non-free \
	>> /etc/apt/sources.list
RUN echo deb-src \
	http://ftp.debian.org/debian/ \
	buster-updates main contrib non-free \
	>> /etc/apt/sources.list
RUN echo deb-src \
	http://security.debian.org/debian-security \
	buster/updates main contrib non-free \
	>> /etc/apt/sources.list
RUN apt-get update
RUN apt-get install -y \
	build-essential \
	pkg-config \
	libopencv-dev \
	libgeoip-dev \
	git wget curl \
	postgresql-client \
	libssl-dev && \
	apt-get clean
RUN apt-get dist-upgrade -y && apt-get clean

# Build newwer libwebp and FFmpeg.
# Using RUN directives caches more readily than a script.
RUN apt-get build-dep -y libwebp ffmpeg && apt-get clean
RUN git clone \
	--branch 1.0.3 \
	--depth 1 \
	https://chromium.googlesource.com/webm/libwebp \
	/src/libwebp
RUN git clone \
	--branch release/4.3 \
	--depth 1 \
	https://github.com/FFmpeg/FFmpeg.git \
	/src/FFmpeg
WORKDIR /src/libwebp
RUN ./autogen.sh
RUN ./configure
RUN nice -n 19 make -j $(nproc)
RUN make install
WORKDIR /src/FFmpeg
RUN ./configure
RUN nice -n 19 make -j $(nproc)
RUN make install

WORKDIR /meguca

# Install Node.js
RUN wget -q -O- https://deb.nodesource.com/setup_10.x | bash -
RUN apt-get install -y nodejs && apt-get clean

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH=$PATH:/root/.cargo/bin

# Build WASM build tools
RUN nice -n 19 cargo install wasm-pack

# Install Go
RUN wget -O- \
	"https://dl.google.com/go/$(curl https://golang.org/VERSION?m=text).linux-amd64.tar.gz" \
	| tar xpz -C /usr/local
ENV PATH=$PATH:/usr/local/go/bin

# Install Go build tools
RUN nice -n 19 go get -u github.com/valyala/quicktemplate \
	github.com/rakyll/statik \
	github.com/valyala/quicktemplate/qtc

# Download go deps
COPY go.mod go.sum ./
RUN go mod download

# Cache Node.js deps
COPY package.json package-lock.json ./
RUN npm install --progress false --depth 0
COPY client/package-lock.json client/package.json client/
RUN cd client && npm install --progress false --depth 0

# Cache Rust dependencies by faking a project structure
RUN mkdir -p \
	client/js client/src www/client \
	websockets/websockets/src \
	protocol/src
COPY Cargo.toml Cargo.lock ./
COPY client/Cargo.toml client/.cargo client/webpack.config.js client/
COPY client/js client/js
COPY docker/dummy.rs client/src/lib.rs
COPY websockets/websockets/Cargo.toml websockets/websockets
COPY docker/dummy.rs websockets/websockets/src/lib.rs
COPY protocol/Cargo.toml protocol
COPY docker/dummy.rs protocol/src/lib.rs
RUN nice -n 19 cargo build --release --workspace --exclude client
RUN cd client && nice -n 19 ./node_modules/.bin/webpack
RUN rm -r \
	client/src websockets/websockets/src protocol/src \
	target/release/deps/libwebsockets* \
	target/release/deps/libprotocol* \
	target/wasm32-unknown-unknown/release/deps/libprotocol* \
	client/dist client/pkg

# Build meguca
COPY . .
RUN NO_DEPS=1 nice -n 19 make
