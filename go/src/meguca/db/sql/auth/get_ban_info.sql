select ip, board, forPost, reason, by, expires
	from bans
	where ip = $1 and board = $2 and expires >= now()
	limit 1
