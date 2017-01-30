insert into bans (ip, board, reason, by, expires)
	values ($1, $2, $3, $4, $5)
	on conflict do nothing
