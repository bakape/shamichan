select op, board, editing, banned, spoiler, deleted, sage, id, time, body, flag,
		name, trip, auth, links, commands, imageName, posterID,
		images.*
	from posts
	left outer join images
		on posts.SHA1 = images.SHA1
	where id = $1
