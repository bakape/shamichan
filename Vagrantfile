Vagrant.configure(2) do |config|
	config.vm.box = "ubuntu/trusty32"
	config.vm.provision "shell", inline: <<-SHELL
		export DEBIAN_FRONTEND=noninteractive

		#Golang PPA
		add-apt-repository ppa:ubuntu-lxc/lxd-stable -y

		# RethinkDB repo
		source /etc/lsb-release
		echo "deb http://download.rethinkdb.com/apt $DISTRIB_CODENAME main" \
			| sudo tee /etc/apt/sources.list.d/rethinkdb.list
	    wget -qO- https://download.rethinkdb.com/apt/pubkey.gpg \
			| sudo apt-key add -

        # Node.js setup script
        wget -q -O - https://deb.nodesource.com/setup_5.x | bash -

    	echo "Installing dependancies..."
        apt-get install -y nodejs build-essential git golang rethinkdb \
			< /dev/null

		# Init RethinkDB
		sudo cp /etc/rethinkdb/default.conf.sample \
			/etc/rethinkdb/instances.d/instance1.conf
		sudo /etc/init.d/rethinkdb restart

		su vagrant -
			cd /vagrant

			echo "Building server..."
			make server

			echo "Building client. This can take a while..."
			make client

			# cd to meguca's root on login
			echo 'cd /vagrant' >> /etc/profile
		exit

		echo "Virtual machine setup finished successfully"
	SHELL

	# Server
	config.vm.network :forwarded_port, host: 8000, guest: 8000
end
