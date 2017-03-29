select * from bans
	where ip = $1 and board = $2 and expires >= now()
