delete from bans
	where board = $1 and forPost = $2
	returning pg_notify('bans_updated', '')
