insert into bans (ip, board, forPost, reason, by, expires)
	values ($1, $2, $3, $4, $5, $6)
	on conflict do nothing
