update posts
	set SHA1 = $2,
		imageName = $3
	where id = $1
