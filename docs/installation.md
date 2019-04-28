List of installation and update commands to set up meguca on Debian stretch.
__Use as a reference. Copy paste at your own risk.__
All commands assume to be run by the root user.

## Install

```bash
# Install OS dependencies
apt update
apt install -y build-essential pkg-config libpth-dev libavcodec-dev libavutil-dev libavformat-dev libswscale-dev libwebp-dev libopencv-dev libgeoip-dev git nodejs
apt-get dist-upgrade -y

# Install PostgreSQL
echo deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -sc)-pgdg main >> /etc/apt/sources.list.d/pgdg.list
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
apt update
apt install -y postgresql

# Increase PostgreSQL connection limit by changing `max_connections` to 1024
sed -i "/max_connections =/d" /etc/postgresql/11/main/postgresql.conf
echo max_connections = 1024 >> /etc/postgresql/11/main/postgresql.conf

# Create users and DBS
service postgresql start
su postgres
psql -c "CREATE USER meguca WITH LOGIN PASSWORD 'meguca' CREATEDB"
createdb -T template0 -E UTF8 -O meguca meguca
exit

# Install Go
wget -O- https://dl.google.com/go/go1.12.4.linux-amd64.tar.gz | tar xpz -C /usr/local
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile

# Install Node.js
wget -qO- https://deb.nodesource.com/setup_10.x | bash -
apt-get install -y nodejs

# Clone and build meguca
git clone https://github.com/bakape/meguca.git meguca
cd meguca
make

# Run meguca
./meguca help
./meguca -a :80
```

## Update

```bash
cd meguca

# Pull changes
git pull

# Rebuild
make

# Restart running instance
./meguca -a :80 restart
```
