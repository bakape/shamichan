#! /bin/bash
set -e

apt-get build-dep -y libwebp ffmpeg

mkdir /src

git clone \
	--branch 1.0.3 \
	--depth 1 \
	https://chromium.googlesource.com/webm/libwebp \
	/src/libwebp
git clone \
	--branch release/4.3 \
	--depth 1 \
	https://github.com/FFmpeg/FFmpeg.git \
	/src/FFmpeg

cd /src/libwebp
./autogen.sh
./configure
nice -n 19 make -j $(nproc)
make install

cd /src/FFmpeg
./configure
nice -n 19 make -j $(nproc)
make install

# Don't include build needless sources in docker image
rm -rf /src
