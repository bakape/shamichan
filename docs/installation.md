List of installation and update commands to set up meguca on Debian jessie.
Use as a reference.

##Install

```bash
# Install RethinkDB
echo "deb http://download.rethinkdb.com/apt `lsb_release -cs` main" | tee /etc/apt/sources.list.d/rethinkdb.list
wget -qO- https://download.rethinkdb.com/apt/pubkey.gpg | apt-key add -
apt-get update
apt-get install rethinkdb
cp /etc/rethinkdb/default.conf.sample /etc/rethinkdb/instances.d/instance1.conf
service rethinkdb start

# Install Node.js
wget -qO- https://deb.nodesource.com/setup_7.x | bash -
apt-get install -y nodejs

# Install Go
wget -O- https://storage.googleapis.com/golang/go1.7.4.linux-amd64.tar.gz | tar xpz -C /usr/local
echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile
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

##update

```bash
cd /meguca

# Pull changes
git pull
git tag
git checkout v3.1.0

# Rebuild
make update_deps all

# Restart running instance
./meguca -a :80 restart
```
