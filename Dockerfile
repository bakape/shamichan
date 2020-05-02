FROM golang

EXPOSE 8000

RUN mkdir -p /meguca/images
ENTRYPOINT ["./scripts/with_postgres.sh"]
CMD ["./meguca", "-a", ":8000"]
WORKDIR /meguca

# Install OS dependencies
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update
RUN apt-get install -y \
	build-essential \
	pkg-config \
	libpth-dev \
	libavcodec-dev libavutil-dev libavformat-dev libswscale-dev \
	libwebp-dev \
	libopencv-dev \
	libgeoip-dev \
	git lsb-release wget curl netcat postgresql-client \
	libssl-dev
RUN apt-get dist-upgrade -y

# Compile newer FFmpeg and deps
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
RUN apt-get build-dep -y libwebp ffmpeg
RUN mkdir /src
RUN git clone \
	--branch 1.0.3 \
	--depth 1 \
	https://chromium.googlesource.com/webm/libwebp \
	/src/libwebp
RUN git clone \
	--branch release/4.2 \
	--depth 1 \
	https://github.com/FFmpeg/FFmpeg.git \
	/src/FFmpeg
WORKDIR /src/libwebp
RUN ./autogen.sh
RUN ./configure
RUN make -j $(nproc)
RUN make install
WORKDIR /src/FFmpeg
RUN ./configure
RUN make -j $(nproc)
RUN make install
WORKDIR /meguca

# Install Node.js
RUN wget -q -O- https://deb.nodesource.com/setup_10.x | bash -
RUN apt-get install -y nodejs

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH=$PATH:/root/.cargo/bin

# Cache config update
COPY docs/config.json .
RUN sed -i 's/localhost:5432/postgres:5432/' config.json
RUN sed -i 's/"reverse_proxied": false/"reverse_proxied": true/' config.json
RUN sed -i 's/127\.0\.0\.1:8000/:8000/' config.json

# Cache dependencies, if possible
RUN cargo install wasm-pack
RUN go get -u github.com/valyala/quicktemplate \
	github.com/rakyll/statik \
	github.com/valyala/quicktemplate/qtc
COPY go.mod go.sum ./
RUN go mod download
COPY package.json package-lock.json ./
RUN npm install

# Copy and build meguca
COPY . .
RUN make
