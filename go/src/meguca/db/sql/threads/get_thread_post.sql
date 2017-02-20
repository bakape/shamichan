select editing, banned, spoiler, id, time, body, name, trip, auth, links,
		backlinks, commands, imageName, images.*
	from posts
	left outer join images
		on posts.SHA1 = images.SHA1
	where id = $1 and (deleted is null or deleted = 'false')
