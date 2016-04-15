#!/usr/bin/env bash
# Automatic deploymnent script for a fresh Ubuntu trusty (and possibly up) VPS

export DEBIAN_FRONTEND=noninteractive;
apt-get update  < /dev/null
apt-get install -y build-essential git redis-server imagemagick\
	iptables-persistent	software-properties-common pngquant < /dev/null
# ffmpeg PPA
add-apt-repository ppa:mc3man/trusty-media -y
# Node.js setup script
wget -q -O - https://deb.nodesource.com/setup_5.x | bash -
apt-get dist-upgrade -y  < /dev/null
apt-get install ffmpeg nodejs -y  < /dev/null

# Port redirection
iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8000;
iptables-save > /etc/iptables/rules.v4

useradd -d /home/meguca -s /bin/bash -m -U meguca
# Download and build meguca
su - meguca << 'EOF'
	cd ~
	git clone -b stable https://github.com/bakape/meguca.git server
	cd server
	# Checkout latest stable version
	git checkout `git tag | tail -n 1`
	npm i
	# cd to meguca's root on login
	echo 'cd ~/server' >> ~/.bashrc
EOF

echo '
User "meguca" has been created. Use "su meguca" as root to switch to this
user and access your meguca root directory. When inside, you can use
"npm start" "npm stop" and "npm restart" to accordingly start/stop/restart
the meguca server. Meguca will become available from "http://${your_IP}". Take
time to learn basic Linux commands and configure the text files in ./config/
later.'
exit
