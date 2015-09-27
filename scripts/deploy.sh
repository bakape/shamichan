#!/usr/bin/env bash
# Automatic deploymnent script for a fresh Ubuntu trusty (and possibly up) VPS

apt-get update
apt-get install -y build-essential git redis-server imagemagick\
	software-properties-common pngquant
add-apt-repository ppa:mc3man/trusty-media -y
wget -q -O - https://deb.nodesource.com/setup_0.12 | bash -
apt-get update
apt-get dist-upgrade -y
apt-get install ffmpeg nodejs -y

useradd -d /home/meguca -s /bin/bash -m -U meguca

# Port redirection
iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8000

su meguca -
	cd ~
	git clone https://github.com/bakape/meguca.git server
	cd server
	git checkout `git tag | tail -n 1`
	npm i
	# cd to meguca's root on login
	echo 'cd ~/server' >> ~/.bashrc
exit

echo 'User "meguca" has been created. Use "su meguca" as root to switch to
this user and access your meguca root directory. When inside, you can use
"npm start" "npm stop" and "npm restart" to accordingly start/stop/restart
the meguca server. Meguca will become available from "http://${your_IP}". Take
time to learn basic Linux commands and configure the text files in ./config/
later.'
exit
