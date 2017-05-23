select id, deleted, banned from posts
	where op = $1 and (deleted = true or banned = true)
