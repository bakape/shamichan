select ip, board from bans
	where expires >= now() at time zone 'utc'
