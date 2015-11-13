Vagrant.configure(2) do |config|
	config.vm.box = "ubuntu/trusty32"
	config.vm.provision "shell", inline: <<-SHELL
		echo "Updating virtual machine..."
		export DEBIAN_FRONTEND=noninteractive

        # ffmpeg PPA
        add-apt-repository ppa:mc3man/trusty-media -y

        # Node.js setup script
        wget -q -O - https://deb.nodesource.com/setup_5.x | bash -
        apt-get dist-upgrade -y  < /dev/null

    	echo "Installing dependancies..."
        apt-get install -y ffmpeg nodejs build-essential redis-server\
        	software-properties-common imagemagick pngquant < /dev/null

		echo "Installing npm modules. This will take a while..."
		su vagrant -
			cd /vagrant
			npm i --unsafe-perm

			# cd to meguca's root on login
			echo 'cd /vagrant' >> /etc/profile
		exit
	SHELL

	# Server
	config.vm.network :forwarded_port, host: 8000, guest: 8000

	# Node debug port
	config.vm.network :forwarded_port, host: 5858, guest: 5858
end
