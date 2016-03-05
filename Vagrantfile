Vagrant.configure(2) do |config|
	config.vm.box = "ubuntu/trusty32"
	config.vm.provision "shell", inline: <<-SHELL
		export DEBIAN_FRONTEND=noninteractive

		add-apt-repository ppa:ubuntu-lxc/lxd-stable -y

        # Node.js setup script
        wget -q -O - https://deb.nodesource.com/setup_5.x | bash -

    	echo "Installing dependancies..."
        apt-get install -y nodejs build-essential git golang < /dev/nullx

		echo "Compiling project..."
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
