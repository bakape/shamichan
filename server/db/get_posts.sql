SELECT p.num, p.name, p.trip, p.email, p.body, p.parent,
		EXTRACT(epoch FROM p.created) * 1000 AS post_time,
		i.id, i.md5, i.filesize, i.ext, i.width, i.height,
		i.thumb_width, i.thumb_height, i.pinky_width, i.pinky_height,
		p.image_filename,
		EXTRACT(epoch FROM i.created) * 1000 AS image_time
	FROM {DB_POST_TABLE} as p
	LEFT JOIN {DB_IMAGE_TABLE} as i ON p.image = i.id
	WHERE p.parent IS {.section posts_only}NOT{.end} NULL
	ORDER BY p.parent ASC, p.num ASC;
