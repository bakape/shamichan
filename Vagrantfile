Vagrant.configure(2) do |config|
	config.vm.box = "ubuntu/trusty32"
	config.vm.provision "shell", inline: <<-SHELL
		apt-get update || exit
		apt-get install -y redis-server build-essential libpng12-dev imagemagick || exit

		# Install io.js v1.2.0
		echo 'Installing io.js'
		wget -q -O - https://iojs.org/dist/v1.2.0/iojs-v1.2.0-linux-x86.tar.gz | tar xz -C /usr/local --strip=1 || exit
		ln -s /usr/local/bin/node /usr/bin/

		# cd to meguca's root on login
		echo 'cd /vagrant' >> /etc/profile
		
		su vagrant -
			# Bootstrap meguca
			cd /vagrant
			cp config.js.example config.js
			cp hot.js.example hot.js
			cp imager/config.js.example imager/config.js
			cp report/config.js.example report/config.js
			npm install --no-bin-links --unsafe-perm
		exit
	SHELL
	config.vm.network :forwarded_port, host: 8000, guest: 8000
	config.vm.network :forwarded_port, host: 8002, guest: 8002
end
