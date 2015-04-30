#!/bin/bash

# Create config files, if none present
function copy() {
	local file=config/$1
	if [[ ! -a $file ]]; then
		cp config/examples/$1 $file
	fi
} 

copy index.js
copy hot.js
copy imager.js
copy report.js
