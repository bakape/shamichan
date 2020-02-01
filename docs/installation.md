List of installation and update commands to set up meguca on Debian buster.
__Use as a reference. Copy paste at your own risk.__
All commands assume to be run by the root user.

## Install

```bash
# Install OS dependencies
apt update
apt-get install -y build-essential pkg-config libpth-dev libavcodec-dev libavutil-dev libavformat-dev libswscale-dev libwebp-dev libopencv-dev libgeoip-dev git lsb-release wget curl sudo postgresql libssl-dev
apt-get dist-upgrade -y

# Create users and DBS
service postgresql start
su postgres
psql -c "CREATE USER meguca WITH LOGIN PASSWORD 'meguca' CREATEDB"
createdb -T template0 -E UTF8 -O meguca meguca
exit

# Install Go
wget -O- https://dl.google.com/go/go1.13.1.linux-amd64.tar.gz | tar xpz -C /usr/local
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile

# Install Node.js
wget -qO- https://deb.nodesource.com/setup_10.x | bash -
apt-get install -y nodejs

# Clone and build meguca
git clone https://github.com/bakape/meguca.git meguca
cd meguca
make

# Edit instance configs
cp docs/config.json .
nano config.json

# Run meguca
./meguca
```

## Update

```bash
cd meguca

# Pull changes
git pull

# Rebuild
make

# Restart running meguca instance.
# This step depends on how your meguca instance is being managed.
#
# A running meguca instance can be gracefully reloaded by sending it the USR2
# signal.
```
