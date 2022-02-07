List of installation and update commands to set up shamichan on Debian buster.
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
psql -c "CREATE USER shamichan WITH LOGIN PASSWORD 'shamichan' CREATEDB"
createdb -T template0 -E UTF8 -O shamichan shamichan
exit

# Install Go
wget -O- wget "https://dl.google.com/go/$(curl https://golang.org/VERSION?m=text).linux-amd64.tar.gz" | tar xpz -C /usr/local
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile

# Install Node.js
wget -qO- https://deb.nodesource.com/setup_10.x | bash -
apt-get install -y nodejs

# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Clone and build shamichan
git clone https://github.com/bakape/shamichan.git shamichan
cd shamichan
make

# Edit instance configs
cp docs/config.json .
nano config.json

# Run shamichan
./shamichan
```

## Update

```bash
cd shamichan

# Pull changes
git pull

# Rebuild
make

# Restart running shamichan instance.
# This step depends on how your shamichan instance is being managed.
#
# A running shamichan instance can be gracefully reloaded by sending it the USR2
# signal.
```
