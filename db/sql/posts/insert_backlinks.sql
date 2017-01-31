update posts
	set backlinks = backlinks || $2
	where id = $1
