select op, board, editing, banned, spoiler, deleted, id, time, body, name, trip, auth,
		links, backlinks, commands, imageName, images.*
	from posts
	left outer join images
		on posts.SHA1 = images.SHA1
	where id = $1
