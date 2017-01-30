update posts
	set banned = true
	where id = $1 and board = $2
	returning ip, op
