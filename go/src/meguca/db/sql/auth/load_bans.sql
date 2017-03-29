select ip, board from bans
	where expires >= now()
