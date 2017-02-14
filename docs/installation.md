List of installation and update commands to set up meguca on Debian jessie.
Use as a reference. Copy paste at your own risk.
All commands assume to be run by the root user.

##Install

```bash
# Install Node.js
wget -qO- https://deb.nodesource.com/setup_7.x | bash -
apt-get install -y nodejs

# Install and init PostgreSQL
echo deb http://apt.postgresql.org/pub/repos/apt/ jessie-pgdg main >> /etc/apt/sources.list.d/pgdg.list
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
apt-get update
apt-get install -y postgresql
service postgresql start
su postgres
createuser -P meguca
createdb -T template0 -E UTF8 -O meguca meguca
exit

# Install Go
wget -O- https://storage.googleapis.com/golang/go1.7.5.linux-amd64.tar.gz | tar xpz -C /usr/local
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile

# Install C dependencies
apt-get dist-upgrade -y
apt-get install -y build-essential pkg-config libpth-dev libavcodec-dev libavutil-dev libavformat-dev libgraphicsmagick1-dev git zip

# Clone and build meguca
git clone -b v3.1.0 https://github.com/bakape/meguca.git /meguca
cd /meguca
make

# Run meguca
./meguca help
./meguca -a :80
```

##Update

```bash
cd /meguca

# Pull changes
git pull
git tag
git checkout v4.0.0

# Rebuild
make update_deps all

# Restart running instance
./meguca -a :80 restart
```
