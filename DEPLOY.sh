#!/bin/bash
# Basic doushio deployment script. Currently only supports Ubuntu 
# Must be run as root

# Settings
export DOUSHIO_USER=doushio
export DOUSHIO_HOME=/home/$DOUSHIO_USER/
export DOUSHIO_DIR=$DOUSHIO_HOME/server/
export WWW=$DOUSHIO_DIR/www/

# Check distro
. /etc/lsb-release
if [[ $DISTRIB_ID = Ubuntu ]]; then
	function installer(){
		if [[ -z `dpkg-query -l $1 | grep "i  $1"` ]]; then
			echo -e "\e[31mInstalling $1\e[0m"
			apt-get install $1 -y
			# Failed installation
			if [[ -z `dpkg-query -l $1 | grep "i  $1"`  && -z $2 ]]; then
				echo -e "\e[31mFailed to install $1\e[0m"
				exit
			fi
		fi
	}
else
	echo -e "\e[31mFailed to identify distribution or distribution not supported by script\e[0m"
	exit
fi

# Choose and clone repository
function ask_repo(){
	cd $DOUSHIO_HOME
	read -r -p "Which repository would you like to install? 1) lalcmellkmal/doushio (main repository); 2) bakape/doushio (fork) [1/2] "
	if [[ $REPLY = '1' ]]; then
		REPO='lalcmellkmal'
	elif [[ $REPLY = '2' ]]; then
		REPO='bakape'
	else
		echo -e "\e[31mInvalid answer\e[0m"
		ask_repo
		# Prevent duplicate `git clone`
		exit
	fi
	echo -e "\e[31mCloning git repository into $DOUSHIO_DIR\e[0m"
	git clone https://github.com/${REPO}/doushio.git server
	mkdir -p $DOUSHIO_HOME/assets/kana
}
export -f ask_repo

# Initialise doushio for production
function initialise(){
	cd $DOUSHIO_DIR
	echo -e "\e[31mConfiguring doushio for basic operation\e[0m"
	cp config.js.example config.js
	sed -i '/DEBUG/c\	DEBUG: false,' config.js
	sed -i '/SERVE_STATIC_FILES/c\	SERVE_STATIC_FILES: true,' config.js
	sed -i '/SERVE_IMAGES/c\	SERVE_IMAGES: true,' config.js
	sed -i '/GZIP/c\	GZIP: true,' config.js
	cp -f hot.js.example hot.js
	cp -f imager/config.js.example imager/config.js
	cp -f report/config.js.example report/config.js
	echo -e "\e[31mInstalling node modules\e[0m"
	npm install
}
export -f initialise

function install_ffmpeg(){
	read -r -p "Would you like to install ffmpeg? Required for WebM support. Might take a while. [y/n] "
	if [[ $REPLY =~ ^[Yy]$ ]]; then
		echo -e "\e[31mInstalling ffmpeg build dependencies\e[0m"
		installer libvpx-dev
		installer libvorbis-dev
		installer yasm
		su doushio -c -l compile_ffmpeg
		cd ${DOUSHIO_HOME}/ffmpeg
		make install
	fi
}

function compile_ffmpeg(){
	cd $DOUSHIO_HOME
	echo -e "\e[31mCloning ffmpeg repository\e[0m"
	git clone git://source.ffmpeg.org/ffmpeg.git ffmpeg
	cd ${DOUSHIO_HOME}/ffmpeg
	echo -e "\e[31mCompyling ffmpeg\e[0m"
	./configure --enable-libvorbis --enable-libvpx
	make -j`nproc`
}
export -f compile_ffmpeg

function install_init_script(){
	read -r -p "Would you like for doushio to autotstart on server boot? [y/n] "
	if [[ $REPLY =~ ^[Yy]$ ]]; then
		cp $DOUSHIO_DIR/docs/doushio.initscript.example /etc/init.d/doushio
		chmod 755 /etc/init.d/doushio
		# Autostart on runlevel 2
		ln -s /etc/init.d/doushio /etc/rc2.d/
		echo -e "\e[31mUse '/etc/init.d/doushio [start|stop]' as root to start/stop the doushio server\e[0m"
	fi
}

# Create user
if ! id -u $DOUSHIO_USER >/dev/null; then
	echo -e "\e[31mCreating doushio user\e[0m"
	useradd -m -s /bin/bash -U $DOUSHIO_USER
fi
# Install hard dependencies
installer git
installer imagemagick
installer nodejs-legacy
installer npm
installer redis-server
installer libpng-dev true
installer pkg-config
installer build-essential
# Fetch doushio
if [[ ! -a $DOUSHIO_DIR ]]; then
	su doushio -c -l ask_repo
fi
# Initialise server folder for production
su doushio -c -l initialise
if [[ -z `which ffmpeg` ]]; then
	install_ffmpeg
fi
if [[ ! -a /etc/init.d/doushio ]]; then
	install_init_script
fi
# Test start doushio
echo -e "\e[31mTesting doushio server. Doushio should now be accessible from http://your-IP-address:8000 and/or http://your-domain:8000. Use Ctrl+C to exit.\e[0m"
su doushio -c -l "
	cd $DOUSHIO_DIR
	node builder.js
"
echo -e "\e[31mNow configure doushio by editing the config.js, hot.js, imager/config.js and report/config.js files\e[0m"
exit
