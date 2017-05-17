insert into bans (ip, board, forPost, reason, by, expires)
	values ($1, $2, $3, $4, $5, $6)
	on conflict do nothing
	returning log_moderation(
		0::smallint,
		$2::varchar(3),
		$3::bigint,
		$5::varchar(20)
	)
