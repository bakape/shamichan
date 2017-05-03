select id from posts
	where
		ip = (
			select ip from posts
				where id = $1 and board = $2
		)
		and board = $2
