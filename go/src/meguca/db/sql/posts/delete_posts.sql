update posts
	set deleted = true
	where id = ANY($1) and board = $2
