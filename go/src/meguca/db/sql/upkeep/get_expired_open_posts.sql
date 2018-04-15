select id, op, board from posts
	where editing = true
		and time < floor(extract(epoch from now() at time zone 'utc')) - 900
