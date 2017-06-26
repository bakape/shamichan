select posts.id, bumpTime, postCtr, posts.deleted
	from threads
	inner join posts on threads.id = posts.id
