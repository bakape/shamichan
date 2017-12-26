select id, op, board from posts
	where editing = true
		and time < floor(extract(epoch from now())) - 900
