FROM golang:buster

EXPOSE 8000

RUN mkdir -p /meguca/images /meguca/www/videos
CMD ["./meguca", "-a", ":8000"]
WORKDIR /meguca

# Install OS dependencies
ENV DEBIAN_FRONTEND=noninteractive
RUN echo deb-src \
	http://ftp.debian.org/debian/ \
	stable main contrib non-free \
	>> /etc/apt/sources.list
RUN echo deb-src \
	http://ftp.debian.org/debian/ \
	stable-updates main contrib non-free \
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

# Compile newer FFmpeg and deps.
# Put inside script to not produce intermediate containers with dep source code
# and build artefacts.
COPY docker/build_ffmpeg.sh .
RUN ./build_ffmpeg.sh
RUN rm build_ffmpeg.sh

# Install Node.js
RUN wget -q -O- https://deb.nodesource.com/setup_10.x | bash -
RUN apt-get install -y nodejs && apt-get clean

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH=$PATH:/root/.cargo/bin

# Build compilers and preprocessors
RUN nice -n 19 cargo install wasm-pack
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
# Put inside script to not produce intermediate containers with unneeded
# artefacts.
COPY ./docker/build_rust_deps.sh .
RUN ./build_rust_deps.sh
RUN rm ./build_rust_deps.sh

# Build meguca
COPY . .
RUN NO_DEPS=1 nice -n 19 make
