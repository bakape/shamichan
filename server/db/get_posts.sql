SELECT p.num, p.name, p.trip, p.email, p.body, p.parent,
		EXTRACT(epoch FROM p.created) * 1000,
		i.id, i.md5, i.filesize, i.ext, i.width, i.height,
		p.image_filename,
		EXTRACT(epoch FROM i.created) * 1000
	FROM {DB_POST_TABLE} as p
	LEFT JOIN {DB_IMAGE_TABLE} as i ON p.image = i.id
	WHERE p.parent IS {.section posts_only}NOT{.end} NULL
	ORDER BY p.parent ASC, p.num ASC;
