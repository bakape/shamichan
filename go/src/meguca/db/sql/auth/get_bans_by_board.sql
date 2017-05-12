select ip, forPost, reason, by, expires
	from bans
	where board = $1
