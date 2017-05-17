select type, id, by, created
	from mod_log
	where board = $1
	order by created desc
