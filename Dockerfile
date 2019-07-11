FROM golang:stretch

EXPOSE 8000

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
	git lsb-release wget curl netcat postgresql-client
RUN apt-get dist-upgrade -y

# Install Node.js
RUN wget -q -O- https://deb.nodesource.com/setup_10.x | bash -
RUN apt-get install -y nodejs

RUN mkdir -p /meguca/images
ENTRYPOINT ["./scripts/with_postgres.sh"]
CMD ["./meguca", "-a", ":8000"]
WORKDIR /meguca

# Cache dependency downloads, if possible
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY package.json .
COPY package-lock.json .
RUN npm install

# Copy and build meguca
COPY . .
COPY docs/config.json .
RUN sed -i 's/localhost:5432/postgres:5432/' config.json
RUN sed -i 's/127\.0\.0\.1:8000/:8000/' config.json
RUN make
