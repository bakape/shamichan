# Built using `make dockerfiles`. DO NOT EDIT!

# TODO: separate server image from imager image
# TODO: coverage report in the CI
FROM ubuntu:focal

EXPOSE 8000

RUN mkdir -p /meguca/images /meguca/www/videos /src
WORKDIR /meguca
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
	libssl-dev \
	&& apt-get clean
RUN apt-get dist-upgrade -y && apt-get clean

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
	common/src
COPY Cargo.toml Cargo.lock ./
COPY client/Cargo.toml client/.cargo client/webpack.config.js client/
COPY client/js client/js
COPY docker/dummy.rs client/src/lib.rs
COPY websockets/websockets/Cargo.toml websockets/websockets
COPY docker/dummy.rs websockets/websockets/src/lib.rs
COPY common/Cargo.toml common
COPY docker/dummy.rs common/src/lib.rs
RUN nice -n 19 cargo build --release --workspace --exclude client
RUN cd client && nice -n 19 ./node_modules/.bin/webpack
RUN rm -r \
	client/src websockets/websockets/src common/src \
	target/release/deps/libwebsockets* \
	target/release/deps/libcommon* \
	target/wasm32-unknown-unknown/release/deps/libcommon* \
	client/dist client/pkg

# Copy all sources
COPY . .



RUN NO_DEPS=1 nice -n 19 make

################################################################################

FROM ubuntu::focal

RUN mkdir -p /meguca/images /meguca/www/videos
WORKDIR /meguca
CMD ["-a", ":8000"]
ENTRYPOINT ["./meguca"]

# Install OS dependencies
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update
RUN apt-get install -y \
	libopencv-dev \
	libgeoip-dev \
	libssl-dev && \
	apt-get clean
RUN apt-get dist-upgrade -y && apt-get clean

# Copy compiled files from dev image
COPY --from=0 /meguca/meguca /meguca/meguca
COPY --from=0 /meguca/www /meguca/ww
