#!/bin/bash

# Create config files, if none present
function copy() {
	if [[ ! -a $1 ]]; then
		cp ${1}.example $1
	fi
} 

copy config.js
copy hot.js
copy imager/config.js
copy report/config.js
