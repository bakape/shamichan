select posts.id, bumpTime,
		(
			select count(*)
			from posts
			where posts.op = threads.id
		) as postCtr,
		posts.deleted
	from threads
	inner join posts on threads.id = posts.id
